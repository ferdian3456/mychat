package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5"
	"github.com/julienschmidt/httprouter"
	"go.uber.org/zap"
	"mychat/helper"
	"mychat/model"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(request *http.Request) bool {
		return true
	},
}
var connections = make(map[string]*websocket.Conn)

func (h *Handler) WebSocket(writer http.ResponseWriter, request *http.Request) {
	ctx := request.Context()
	userUUID, _ := ctx.Value("user_uuid").(string)

	conn, err := upgrader.Upgrade(writer, request, nil)
	if err != nil {
		// Can't even upgrade — just return
		http.Error(writer, "WebSocket upgrade failed", http.StatusInternalServerError)
		return
	}

	// Save and clean up connection
	connections[userUUID] = conn
	defer func() {
		conn.Close()
		delete(connections, userUUID)
	}()

	for {
		// Reset validation errors on every message
		errorMap := map[string]string{}

		_, data, err := conn.ReadMessage()
		if err != nil {
			fmt.Println("Error reading message:", err)
			break // break on client disconnect or read failure
		}

		var msg model.IncomingMessage
		err = json.Unmarshal(data, &msg)
		if err != nil {
			errorMap["internal"] = "invalid JSON format"
		}

		if msg.Text == "" {
			errorMap["text"] = "message text cannot be empty"
		}
		if msg.ConversationID <= 0 {
			errorMap["conversation"] = "invalid conversation id"
		}

		if len(errorMap) > 0 {
			errMsg, _ := json.Marshal(map[string]any{
				"type":   "error",
				"errors": errorMap,
			})
			_ = conn.WriteMessage(websocket.TextMessage, errMsg)
			continue
		}

		// Message is valid, insert into DB
		createdAt := time.Now()
		var id int
		query := "INSERT INTO messages (conversation_id, sender_id, text, created_at) VALUES ($1, $2, $3, $4) RETURNING id"
		err = h.Config.DB.QueryRow(ctx, query, msg.ConversationID, userUUID, msg.Text, createdAt).Scan(&id)
		if err != nil {
			h.Config.Log.Error("failed to insert message", zap.Error(err))
			continue // do not panic, just skip this one
		}

		stored := model.Message{
			ID:             id,
			ConversationID: msg.ConversationID,
			SenderID:       userUUID,
			Text:           msg.Text,
			CreatedAt:      createdAt,
		}

		// Broadcast to all participants
		query = "SELECT user_id FROM conversation_participants WHERE conversation_id = $1"
		rows, err := h.Config.DB.Query(ctx, query, msg.ConversationID)
		if err != nil {
			h.Config.Log.Error("failed to query participants", zap.Error(err))
			continue
		}
		func() {
			defer rows.Close()
			for rows.Next() {
				var participantID string
				if err := rows.Scan(&participantID); err == nil {
					if participantConn, ok := connections[participantID]; ok {
						_ = participantConn.WriteJSON(stored) // ignore write errors
					}
				}
			}
		}()
	}
}

func (h *Handler) GetWebsocketToken(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
	ctx := request.Context()
	userUUID := ctx.Value("user_uuid").(string)

	// Generate UUID
	wsToken := uuid.New().String()

	duration := 5 * time.Minute
	durationInSecond := int(duration.Seconds())

	err := h.Config.DBCache.Set(ctx, "ws_token:"+wsToken, userUUID, 5*time.Minute).Err()
	if err != nil {
		h.Config.Log.Panic("failed to set key in redis db", zap.Error(err))
	}

	response := model.WebsocketTokenResponse{
		WebsocketToken:          wsToken,
		TokenType:               "opaque",
		WebsocketTokenExpiresIn: durationInSecond,
	}

	helper.WriteSuccessResponse(writer, response)
}

func (h *Handler) GetMessage(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
	ctx := request.Context()

	conversationID, _ := strconv.Atoi(params.ByName("id"))
	beforeIDStr := request.URL.Query().Get("before_id")
	limitStr := request.URL.Query().Get("limit")

	limit := 20
	l, err := strconv.Atoi(limitStr)
	if err == nil && l > 0 {
		limit = l
	}

	var rows pgx.Rows

	if beforeIDStr != "" {
		beforeID, _ := strconv.Atoi(beforeIDStr)
		query := "SELECT id, sender_id, text, created_at FROM messages WHERE conversation_id = $1 AND id < $2 ORDER BY id DESC LIMIT $3"
		rows, err = h.Config.DB.Query(ctx, query, conversationID, beforeID, limit)
	} else {
		query := "SELECT id, sender_id, text, created_at FROM messages WHERE conversation_id = $1 ORDER BY id DESC LIMIT $2"
		rows, err = h.Config.DB.Query(ctx, query, conversationID, limit)
	}

	if err != nil {
		h.Config.Log.Panic("failed to query into database", zap.Error(err))
	}

	defer rows.Close()

	var messages []model.Message
	for rows.Next() {
		var m model.Message
		m.ConversationID = conversationID
		err = rows.Scan(&m.ID, &m.SenderID, &m.Text, &m.CreatedAt)
		if err == nil {
			messages = append(messages, m)
		}
	}

	helper.WriteSuccessResponse(writer, messages)
}

func (h *Handler) CreateConversation(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
	ctx := request.Context()
	userUUID := ctx.Value("user_uuid").(string)

	var payload model.UserConversationRequest
	helper.ReadFromRequestBody(request, &payload)

	// block if the user includes themselves manually in json payload
	for _, id := range payload.ParticipantIDs {
		if id == userUUID {
			helper.WriteErrorResponse(writer, http.StatusBadRequest, map[string]string{
				"participant_ids": "you should not include yourself in the participant list",
			})
			return
		}
	}

	// ensure the current user is included
	participantSet := map[string]struct{}{userUUID: {}}
	for _, id := range payload.ParticipantIDs {
		participantSet[id] = struct{}{}
	}

	// flatten to sorted slice
	allParticipants := make([]string, 0, len(participantSet))
	for id := range participantSet {
		allParticipants = append(allParticipants, id)
	}
	sort.Strings(allParticipants)

	// Step 1: Verify all participant IDs exist
	query := "SELECT id FROM users WHERE id = ANY($1)"
	rows, err := h.Config.DB.Query(ctx, query, allParticipants)
	if err != nil {
		h.Config.Log.Panic("failed to query users", zap.Error(err))
	}
	defer rows.Close()

	foundUsers := map[string]bool{}
	for rows.Next() {
		var id string
		_ = rows.Scan(&id)
		foundUsers[id] = true
	}

	missing := []string{}
	for _, id := range allParticipants {
		if !foundUsers[id] {
			missing = append(missing, id)
		}
	}

	if len(missing) > 0 {
		helper.WriteErrorResponse(writer, http.StatusBadRequest, map[string]string{
			"missing_participants": strings.Join(missing, ", "),
		})
		return
	}

	// Step 2: Check if a conversation with exact same users exists
	query = `
	SELECT cp.conversation_id
	FROM conversation_participants cp
	WHERE cp.user_id = ANY($1)
	GROUP BY cp.conversation_id
	HAVING COUNT(*) = $2
	   AND COUNT(*) = (
		 SELECT COUNT(*) FROM conversation_participants WHERE conversation_id = cp.conversation_id
	   )
	LIMIT 1;
	`

	var conversationID int
	err = h.Config.DB.QueryRow(ctx, query, allParticipants, len(allParticipants)).Scan(&conversationID)
	if err == pgx.ErrNoRows {
		// Step 3: Create new conversation
		err = h.Config.DB.QueryRow(ctx, "INSERT INTO conversations (created_at) VALUES (NOW()) RETURNING id").Scan(&conversationID)
		if err != nil {
			h.Config.Log.Panic("failed to insert conversation", zap.Error(err))
		}

		// Step 4: Add participants
		batch := &pgx.Batch{}
		for _, id := range allParticipants {
			batch.Queue("INSERT INTO conversation_participants (conversation_id, user_id) VALUES ($1, $2)", conversationID, id)
		}
		br := h.Config.DB.SendBatch(ctx, batch)
		defer br.Close()
		for i := 0; i < len(allParticipants); i++ {
			if _, err := br.Exec(); err != nil {
				h.Config.Log.Panic("failed to insert participant", zap.Error(err))
			}
		}
	} else if err != nil {
		h.Config.Log.Panic("failed to query existing conversation", zap.Error(err))
	}

	response := model.UserConversationResponse{
		ConversationID: conversationID,
	}
	// ✅ Done — return conversation ID
	helper.WriteSuccessResponse(writer, response)
}

func (h *Handler) GetParticipantInfo(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
	ctx := request.Context()

	errorMap := map[string]string{}
	userUUID, _ := ctx.Value("user_uuid").(string)
	conversationID := params.ByName("id")

	// start transaction
	tx, err := h.Config.DB.Begin(ctx)
	if err != nil {
		h.Config.Log.Panic("failed to start transaction", zap.Error(err))
	}

	defer helper.CommitOrRollback(ctx, tx, h.Config.Log)

	query := "SELECT user_id FROM conversation_participants WHERE conversation_id=$1 AND user_id!=$2"

	var participantID string

	err = tx.QueryRow(ctx, query, conversationID, userUUID).Scan(&participantID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			errorMap["conversation_id"] = "conversation not found"
		} else {
			h.Config.Log.Panic("failed to query into database", zap.Error(err))
		}
	}

	if len(errorMap) > 0 {
		helper.WriteErrorResponse(writer, http.StatusBadRequest, errorMap)
		return
	}

	var participantName string
	query = "SELECT username FROM users WHERE id=$1"
	err = tx.QueryRow(ctx, query, participantID).Scan(&participantName)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			errorMap["user"] = "user not found"
		} else {
			h.Config.Log.Panic("failed to query into database", zap.Error(err))
		}
	}

	if len(errorMap) > 0 {
		helper.WriteErrorResponse(writer, http.StatusBadRequest, errorMap)
		return
	}

	user := model.UserInfoResponse{
		Id:       participantID,
		Username: participantName,
	}

	helper.WriteSuccessResponse(writer, user)
}
