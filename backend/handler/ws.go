package handler

import (
	"encoding/json"
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

var upgrader = websocket.Upgrader{}
var connections = make(map[string]*websocket.Conn)

func (h *Handler) WebSocket(writer http.ResponseWriter, request *http.Request) {
	ctx := request.Context()
	fmt.Println("got here1")

	userUUID, _ := ctx.Value("user_uuid").(string)

	errorMap := map[string]string{}

	fmt.Println("got here-2")

	conn, err := upgrader.Upgrade(writer, request, nil)
	if err != nil {
		errorMap["connection"] = "websocket upgrade error"
		fmt.Println("woi", err)
	}

	if len(errorMap) > 0 {
		errMsg, _ := json.Marshal(map[string]any{
			"type":   "error",
			"errors": errorMap,
		})
		conn.WriteMessage(websocket.TextMessage, errMsg)
	}

	fmt.Println("got here2")

	connections[userUUID] = conn
	defer func() {
		conn.Close()
		delete(connections, userUUID)
	}()

	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			errorMap["internal"] = "error reading websocket message"
			break
		}

		var msg model.IncomingMessage
		err = json.Unmarshal(data, &msg)
		if err != nil {
			errorMap["internal"] = "error unmarshalling json"
			break
		}

		if msg.Text == "" {
			errorMap["text"] = "message text cannot be empty"
		}

		//if msg.SenderID != userUUID {
		//	errorMap["sender"] = "invalid sender"
		//}

		if msg.ConversationID <= 0 {
			errorMap["conversation"] = "invalid conversation id"
		}

		if len(errorMap) > 0 {
			errMsg, _ := json.Marshal(map[string]any{
				"type":   "error",
				"errors": errorMap,
			})
			conn.WriteMessage(websocket.TextMessage, errMsg)
			break
		}

		createdAt := time.Now()
		var id int
		query := "INSERT INTO messages (conversation_id,sender_id, text,created_at) VALUES($1,$2,$3,$4) RETURNING id"
		err = h.Config.DB.QueryRow(ctx, query, msg.ConversationID, userUUID, msg.Text, createdAt).Scan(&id)
		if err != nil {
			h.Config.Log.Panic("failed to query into database", zap.Error(err))
		}

		stored := model.Message{
			ID:             id,
			ConversationID: msg.ConversationID,
			SenderID:       userUUID,
			Text:           msg.Text,
			CreatedAt:      createdAt,
		}

		query = "SELECT user_id FROM conversation_participants WHERE conversation_id = $1"
		rows, err := h.Config.DB.Query(ctx, query, msg.ConversationID)
		if err != nil {
			h.Config.Log.Panic("failed to query into database", zap.Error(err))
		}

		defer rows.Close()

		for rows.Next() {
			var participantId string
			err = rows.Scan(&participantId)
			if err == nil {
				conn, ok := connections[participantId]
				if ok {
					conn.WriteJSON(stored)
				}
			}
		}
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
