package usecase

import (
	"context"
	"github.com/ferdian3456/mychat/backend/chat-service/internal/model"
	"github.com/ferdian3456/mychat/backend/chat-service/internal/repository"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/knadh/koanf/v2"
	"go.uber.org/zap"
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

func (usecase *ChatUsecase) GetAllUserData(ctx context.Context, userUUID string, errorMap map[string]string) ([]model.AllUserInfoResponse, map[string]string) {
	//user, errorMap := usecase.ChatRepository.GetAllUserData(ctx, userUUID, errorMap)
	//if errorMap != nil {
	//	return user, errorMap
	//}
	//
	//return user, nil
	return nil, nil
}
