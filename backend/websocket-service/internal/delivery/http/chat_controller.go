package http

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ferdian3456/mychat/backend/websocket-service/internal/helper"
	"github.com/ferdian3456/mychat/backend/websocket-service/internal/model"
	"github.com/ferdian3456/mychat/backend/websocket-service/internal/usecase"
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
	errorMap := map[string]string{}

	userUUID := ctx.Value("user_uuid").(string)

	var payload model.UserAddConversationRequest
	helper.ReadFromRequestBody(request, &payload)

	response, err := controller.ChatUsecase.CreateConversation(ctx, payload, userUUID, errorMap)
	if err != nil {
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

	connection, err := upgrader.Upgrade(writer, request, nil)
	if err != nil {
		helper.WriteErrorResponse(writer, http.StatusBadRequest, map[string]string{
			"connection": "failed to upgrade websocket",
		})
		return
	}
	defer connection.Close()

	// Assign user to a Redis pubsub bucket
	bucket := helper.GetBucketForUser(userUUID, 1024)
	channel := fmt.Sprintf("deliver:bucket:%d", bucket)

	// Create cancelable context for pubsub listener
	pubsubCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	pubsub := controller.ChatUsecase.SubscribeToBucket(pubsubCtx, channel)
	defer pubsub.Close()

	// Start Redis pubsub listener
	go func() {
		ch := pubsub.Channel()
		for {
			select {
			case <-pubsubCtx.Done():
				return
			case msg, ok := <-ch:
				if !ok {
					return // channel closed
				}
				//fmt.Println(msg.Payload)
				if helper.MessageBelongsToUser(msg.Payload, userUUID) {
					_ = connection.WriteMessage(websocket.TextMessage, []byte(msg.Payload))
				}
			}
		}
	}()

	// WebSocket read loop
	for {
		_, data, err := connection.ReadMessage()
		if err != nil {
			break
		}

		var msg model.IncomingMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			_ = connection.WriteJSON(map[string]any{
				"status": http.StatusText(http.StatusBadRequest),
				"errors": map[string]string{"message": "invalid json format"},
			})
			continue
		}

		if errMap := controller.ChatUsecase.SendMessage(ctx, msg, userUUID); errMap != nil {
			_ = connection.WriteJSON(map[string]any{
				"status": http.StatusText(http.StatusInternalServerError),
				"errors": map[string]string{"message": "failed to send message"},
			})
		}
	}

	cancel() // stop goroutine
}

func (controller ChatController) GetAllMyOwnConversationID(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
	ctx := request.Context()
	userUUID, _ := ctx.Value("user_uuid").(string)

	errorMap := map[string]string{}

	response, errorMap := controller.ChatUsecase.GetAllMyOwnConversationID(ctx, userUUID, errorMap)
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
