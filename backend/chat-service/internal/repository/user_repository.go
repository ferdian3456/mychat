package repository

import (
	"context"
	"github.com/ferdian3456/mychat/backend/chat-service/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type ChatRepository struct {
	Log     *zap.Logger
	DB      *pgxpool.Pool
	DBCache *redis.ClusterClient
}

func NewChatRepository(zap *zap.Logger, db *pgxpool.Pool, dbCache *redis.ClusterClient) *ChatRepository {
	return &ChatRepository{
		Log:     zap,
		DB:      db,
		DBCache: dbCache,
	}
}

func (repository *ChatRepositorys) RegisterWithTx(ctx context.Context, tx pgx.Tx, user model.User, errorMap map[string]string) map[string]string {
	query := "INSERT INTO users (id,username,password,created_at,updated_at) VALUES ($1,$2,$3,$4,$5)"
	_, err := tx.Exec(ctx, query, user.Id, user.Username, user.Password, user.Created_at, user.Updated_at)
	if err != nil {
		errorMap["internal"] = "failed to query into database"
		return errorMap
		//repository.Log.Panic("failed to query into database", zap.Error(err))
	}

	return nil
}
