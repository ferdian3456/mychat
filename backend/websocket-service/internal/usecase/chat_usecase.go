package usecase

import (
	"context"
	"encoding/json"
	"github.com/ferdian3456/mychat/backend/websocket-service/internal/helper"
	"github.com/ferdian3456/mychat/backend/websocket-service/internal/model"
	"github.com/ferdian3456/mychat/backend/websocket-service/internal/repository"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/knadh/koanf/v2"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"sort"
	"strconv"
	"time"
)

type ChatUsecase struct {
	ChatRepository *repository.ChatRepository
	DB             *pgxpool.Pool
	Log            *zap.Logger
	Config         *koanf.Koanf
}

func NewChatUsecase(chatRepository *repository.ChatRepository, db *pgxpool.Pool, zap *zap.Logger, koanf *koanf.Koanf) *ChatUsecase {
	return &ChatUsecase{
		ChatRepository: chatRepository,
		DB:             db,
		Log:            zap,
		Config:         koanf,
	}
}

func (usecase *ChatUsecase) GetMessage(ctx context.Context, conversationID int, beforeIDStr string, limit int, errorMap map[string]string) ([]model.Message, map[string]string) {
	var messages []model.Message

	if beforeIDStr != "" {
		beforeID, _ := strconv.Atoi(beforeIDStr)
		messages, errorMap = usecase.ChatRepository.GetPreviousMessageWithChatID(ctx, conversationID, beforeID, limit, errorMap)
		if errorMap != nil {
			return messages, errorMap
		}
	} else {
		messages, errorMap = usecase.ChatRepository.GetPreviousMessage(ctx, conversationID, limit, errorMap)
		if errorMap != nil {
			return messages, errorMap
		}
	}

	return messages, nil
}

func (usecase *ChatUsecase) CreateConversation(ctx context.Context, payload model.UserConversationRequest, userUUID string, errorMap map[string]string) (model.UserConversationResponse, map[string]string) {
	var conversation model.UserConversationResponse

	for _, id := range payload.ParticipantIDs {
		if id == userUUID {
			errorMap["participant_id"] = "you should not include yourself in the participant list"
			return conversation, errorMap
		}
	}

	// ensure the current user is included into payload
	participantSet := map[string]struct{}{
		userUUID: {},
	}

	for _, id := range payload.ParticipantIDs {
		participantSet[id] = struct{}{}
	}

	allParticipants := make([]string, 0, len(participantSet))
	for id := range participantSet {
		allParticipants = append(allParticipants, id)
	}

	sort.Strings(allParticipants)

	// start transaction
	tx, err := usecase.DB.Begin(ctx)
	if err != nil {
		errorMap["internal"] = "failed to start transaction"
		return conversation, errorMap
	}

	defer helper.CommitOrRollback(ctx, tx, usecase.Log)

	errorMap = usecase.ChatRepository.VerifyAllUserID(ctx, tx, allParticipants, errorMap)
	if errorMap != nil {
		_ = tx.Rollback(ctx)
		return conversation, errorMap
	}

	conversationID, errorMap := usecase.ChatRepository.GetConversationIDByParticipants(ctx, tx, allParticipants, errorMap)
	if errorMap != nil {
		_ = tx.Rollback(ctx)
		return conversation, errorMap
	}

	conversation.ConversationID = conversationID
	return conversation, nil
}

func (usecase *ChatUsecase) GetParticipantInfo(ctx context.Context, userUUID string, conversationID int, errorMap map[string]string) (model.UserInfoResponse, map[string]string) {
	var participant model.UserInfoResponse

	// start transaction
	tx, err := usecase.DB.Begin(ctx)
	if err != nil {
		errorMap["internal"] = "failed to start transaction"
		return participant, errorMap
	}

	defer helper.CommitOrRollback(ctx, tx, usecase.Log)

	participant.Id, errorMap = usecase.ChatRepository.GetParticipantID(ctx, tx, userUUID, conversationID, errorMap)
	if errorMap != nil {
		_ = tx.Rollback(ctx)
		return participant, errorMap
	}

	participant.Username, errorMap = usecase.ChatRepository.GetParticipantName(ctx, tx, participant.Id, errorMap)
	if errorMap != nil {
		_ = tx.Rollback(ctx)
		return participant, errorMap
	}

	return participant, nil
}

func (usecase *ChatUsecase) GetWebSocketToken(ctx context.Context, userUUID string, errorMap map[string]string) (model.WebsocketTokenResponse, map[string]string) {
	duration := 5 * time.Minute
	durationInSecond := int(duration.Seconds())

	wsTokenResponse := model.WebsocketTokenResponse{
		WebsocketToken:          uuid.New().String(),
		TokenType:               "opaque",
		WebsocketTokenExpiresIn: durationInSecond,
	}

	errorMap = usecase.ChatRepository.SetWSToken(ctx, userUUID, wsTokenResponse.WebsocketToken, duration, errorMap)
	if errorMap != nil {
		return wsTokenResponse, errorMap
	}

	return wsTokenResponse, nil
}

//func (usecase *ChatUsecase) ProcessIncomingMessage(ctx context.Context, msg model.IncomingMessage, senderUUID string) (model.Message, map[string]string) {
//	errorMap := map[string]string{}
//
//	message := model.Message{}
//
//	if msg.Text == "" {
//		errorMap["text"] = "text is required to not be empty"
//		return message, errorMap
//	}
//	if msg.ConversationID <= 0 {
//		errorMap["conversation_id"] = "conversation id is required to not be empty"
//		return message, errorMap
//	}
//
//	message.ConversationID = msg.ConversationID
//	message.SenderID = senderUUID
//	message.Text = msg.Text
//	message.CreatedAt = time.Now()
//
//	message.ID, errorMap = usecase.ChatRepository.InsertMessage(ctx, message, errorMap)
//	if errorMap != nil {
//		return message, errorMap
//	}
//
//	return message, nil
//}

func (usecase *ChatUsecase) GetParticipants(ctx context.Context, conversationID int) ([]string, map[string]string) {
	return usecase.ChatRepository.GetParticipants(ctx, conversationID)
}

func (usecase *ChatUsecase) CheckUserExistance(ctx context.Context, userUUID string, errorMap map[string]string) map[string]string {
	err := usecase.ChatRepository.CheckUserExistence(ctx, userUUID, errorMap)
	if err != nil {
		return err
	}

	return nil
}

func (usecase *ChatUsecase) VerifyWsToken(ctx context.Context, wsToken string, errorMap map[string]string) (string, map[string]string) {
	userUUID, errorMap := usecase.ChatRepository.VerifyWsToken(ctx, wsToken, errorMap)
	if errorMap != nil {
		return "", errorMap
	}

	return userUUID, nil
}

func (usecase *ChatUsecase) RegisterUserInConversation(ctx context.Context, userUUID string, conversationID int) {
	usecase.ChatRepository.SAddConversationMember(ctx, userUUID, conversationID)
	usecase.ChatRepository.SetUserSession(ctx, userUUID)
}

//func (usecase *ChatUsecase) ConsumeConversationMessages(ctx context.Context, conversationID int, userUUID string, conn *websocket.Conn) {
//	usecase.ChatRepository.ConsumeConversationMessages(ctx, conversationID, userUUID, conn)
//}

func (usecase *ChatUsecase) SendMessage(ctx context.Context, msg model.IncomingMessage, senderUUID string) map[string]string {
	errorMap := map[string]string{}

	if msg.Text == "" {
		errorMap["text"] = "text is required"
		return errorMap
	}

	if msg.ConversationID == 0 {
		errorMap["conversation_id"] = "conversation_id is required"
		return errorMap
	}

	participantIDs, errorMap := usecase.ChatRepository.GetConversationParticipantsByConversationID(ctx, msg.ConversationID, errorMap)
	if errorMap != nil {
		return errorMap
	}

	message := model.Message{
		ID:             uuid.New().String(),
		ConversationID: msg.ConversationID,
		SenderID:       senderUUID,
		RecipientIDs:   participantIDs,
		Text:           msg.Text,
		CreatedAt:      time.Now(),
	}

	jsonPayload, _ := json.Marshal(message)

	topic := "chat-conversation"
	err := usecase.ChatRepository.ProduceToKafka(ctx, topic, jsonPayload)
	if err != nil {
		errorMap["internal"] = "failed to produce to kafka"
	}
	return errorMap
}

func (usecase *ChatUsecase) SubscribeToBucket(ctx context.Context, channel string) *redis.PubSub {
	return usecase.ChatRepository.SubscribeToRedisChannel(ctx, channel)
}
