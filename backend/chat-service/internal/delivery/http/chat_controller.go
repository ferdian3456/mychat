package http

import (
	"encoding/json"
	"github.com/ferdian3456/mychat/backend/chat-service/internal/helper"
	"github.com/ferdian3456/mychat/backend/chat-service/internal/model"
	"github.com/ferdian3456/mychat/backend/chat-service/internal/usecase"
	"github.com/gorilla/websocket"
	"github.com/julienschmidt/httprouter"
	"github.com/knadh/koanf/v2"
	"go.uber.org/zap"
	"net/http"
	"strconv"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(request *http.Request) bool {
		return true
	},
}
var connections = make(map[string]*websocket.Conn)

type ChatController struct {
	ChatUsecase *usecase.ChatUsecase
	Log         *zap.Logger
	Config      *koanf.Koanf
}

func NewChatController(chatUsecase *usecase.ChatUsecase, zap *zap.Logger, koanf *koanf.Koanf) *ChatController {
	return &ChatController{
		ChatUsecase: chatUsecase,
		Log:         zap,
		Config:      koanf,
	}
}

func (controller ChatController) GetMessage(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
	ctx := request.Context()

	errorMap := map[string]string{}

	conversationID, _ := strconv.Atoi(params.ByName("id"))
	beforeIDStr := request.URL.Query().Get("before_id")
	limitStr := request.URL.Query().Get("limit")

	limit := 20
	l, err := strconv.Atoi(limitStr)
	if err == nil && l > 0 {
		limit = l
	}

	response, errorMap := controller.ChatUsecase.GetMessage(ctx, conversationID, beforeIDStr, limit, errorMap)
	if errorMap != nil {
		if errorMap["internal"] != "" {
			helper.WriteErrorResponse(writer, http.StatusInternalServerError, errorMap)
			return
		} else {
			helper.WriteErrorResponse(writer, http.StatusBadRequest, errorMap)
			return
		}
	}

	helper.WriteSuccessResponse(writer, response)
}

func (controller ChatController) CreateConversation(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
	ctx := request.Context()
	var errorMap map[string]string

	userUUID := ctx.Value("user_uuid").(string)

	var payload model.UserConversationRequest
	helper.ReadFromRequestBody(request, &payload)

	response, errorMap := controller.ChatUsecase.CreateConversation(ctx, payload, userUUID, errorMap)
	if errorMap != nil {
		if errorMap["internal"] != "" {
			helper.WriteErrorResponse(writer, http.StatusInternalServerError, errorMap)
			return
		} else {
			helper.WriteErrorResponse(writer, http.StatusBadRequest, errorMap)
			return
		}
	}

	helper.WriteSuccessResponse(writer, response)
}

func (controller ChatController) GetParticipantInfo(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
	ctx := request.Context()

	errorMap := map[string]string{}
	userUUID, _ := ctx.Value("user_uuid").(string)
	conversationID, _ := strconv.Atoi(params.ByName("id"))

	response, errorMap := controller.ChatUsecase.GetParticipantInfo(ctx, userUUID, conversationID, errorMap)
	if errorMap != nil {
		if errorMap["internal"] != "" {
			helper.WriteErrorResponse(writer, http.StatusInternalServerError, errorMap)
			return
		} else {
			helper.WriteErrorResponse(writer, http.StatusBadRequest, errorMap)
			return
		}
	}

	helper.WriteSuccessResponse(writer, response)
}

func (controller ChatController) GetWebSocketToken(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
	ctx := request.Context()

	var errorMap map[string]string
	userUUID := ctx.Value("user_uuid").(string)

	response, errorMap := controller.ChatUsecase.GetWebSocketToken(ctx, userUUID, errorMap)
	if errorMap != nil {
		if errorMap["internal"] != "" {
			helper.WriteErrorResponse(writer, http.StatusInternalServerError, errorMap)
			return
		} else {
			helper.WriteErrorResponse(writer, http.StatusBadRequest, errorMap)
			return
		}
	}

	helper.WriteSuccessResponse(writer, response)
}

func (controller ChatController) WebSocket(writer http.ResponseWriter, request *http.Request) {
	ctx := request.Context()
	userUUID, _ := ctx.Value("user_uuid").(string)

	var errorMap map[string]string

	connection, err := upgrader.Upgrade(writer, request, nil)
	if err != nil {
		errorMap["connection"] = "failed to upgrade to websocket connection"
		helper.WriteErrorResponse(writer, http.StatusBadRequest, errorMap)
		return
	}

	// Save and clean up connection
	connections[userUUID] = connection
	defer func() {
		connection.Close()
		delete(connections, userUUID)
	}()

	for {
		_, data, err := connection.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				controller.Log.Warn("unexpected websocket close", zap.Error(err))
			} else {
				controller.Log.Warn("websocket closed normally", zap.Error(err))
			}
			break
		}

		var msg model.IncomingMessage
		err = json.Unmarshal(data, &msg)
		if err != nil {
			resp := map[string]any{
				"status": http.StatusText(http.StatusBadRequest),
				"errors": map[string]string{
					"message": "invalid json format",
				},
			}

			connection.WriteJSON(resp)
			continue
		}

		storedMsg, errorMap := controller.ChatUsecase.ProcessIncomingMessage(ctx, msg, userUUID)
		if errorMap != nil {
			if errorMap["internal"] != "" {
				_ = connection.WriteJSON(map[string]any{
					"status": http.StatusText(http.StatusInternalServerError),
					"data":   errorMap,
				})
			} else {
				_ = connection.WriteJSON(map[string]any{
					"status": http.StatusText(http.StatusBadRequest),
					"data":   errorMap,
				})
			}

			continue
		}

		participants, errorMap := controller.ChatUsecase.GetParticipants(ctx, msg.ConversationID)
		if errorMap != nil {
			if errorMap["internal"] != "" {
				_ = connection.WriteJSON(map[string]any{
					"status": http.StatusText(http.StatusInternalServerError),
					"data":   errorMap,
				})
			} else {
				_ = connection.WriteJSON(map[string]any{
					"status": http.StatusText(http.StatusBadRequest),
					"data":   errorMap,
				})
			}

			continue
		}

		for _, pid := range participants {
			if participantConn, ok := connections[pid]; ok {
				_ = participantConn.WriteJSON(storedMsg)
			}
		}
	}
}
