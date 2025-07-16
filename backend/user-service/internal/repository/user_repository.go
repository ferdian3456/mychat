package repository

import (
	"context"
	"errors"
	"github.com/ferdian3456/mychat/backend/user-service/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type UserRepository struct {
	Log     *zap.Logger
	DB      *pgxpool.Pool
	DBCache *redis.ClusterClient
}

func NewUserRepository(zap *zap.Logger, db *pgxpool.Pool, dbCache *redis.ClusterClient) *UserRepository {
	return &UserRepository{
		Log:     zap,
		DB:      db,
		DBCache: dbCache,
	}
}

func (repository *UserRepository) RegisterWithTx(ctx context.Context, tx pgx.Tx, user model.User) error {
	query := "INSERT INTO users (id,username,password,created_at,updated_at) VALUES ($1,$2,$3,$4,$5)"
	_, err := tx.Exec(ctx, query, user.Id, user.Username, user.Password, user.Created_at, user.Updated_at)
	if err != nil {
		return errors.New("failed to query into database")
		//repository.Log.Panic("failed to query into database", zap.Error(err))
	}

	return nil
}

func (repository *UserRepository) CheckUsernameUniqueWithTx(ctx context.Context, tx pgx.Tx, username string) error {
	return nil
}

func (repository *UserRepository) AddRefreshTokenWithTx(ctx context.Context, tx pgx.Tx, refreshtoken model.RefreshToken) error {
	query := "INSERT INTO refresh_tokens (user_id,hashed_refresh_token,created_at,expired_at) VALUES ($1,$2,$3,$4)"
	_, err := tx.Exec(ctx, query, refreshtoken.User_id, refreshtoken.Hashed_refresh_token, refreshtoken.Created_at, refreshtoken.Expired_at)
	if err != nil {
		return errors.New("failed to query into database")
		//repository.Log.Panic("failed to query into database", zap.Error(err))
	}

	return nil
}

func (repository *UserRepository) UpdateRefreshTokenWithTx(ctx context.Context, tx pgx.Tx, tokenStatus string, userUUID string) error {
	query := "UPDATE refresh_tokens SET status = $1 WHERE user_id = $2 AND created_at = (SELECT MAX(created_at) FROM refresh_tokens WHERE user_id = $2)"
	_, err := tx.Exec(ctx, query, tokenStatus, userUUID)
	if err != nil {
		return errors.New("failed to query into database")
		//repository.Log.Panic("failed to query into database", zap.Error(err))
	}

	return nil
}

func (repository *UserRepository) LoginWithTx(ctx context.Context, tx pgx.Tx, username string) (model.User, error) {
	query := "SELECT id,password FROM users WHERE username=$1"

	var user model.User
	err := tx.QueryRow(ctx, query, username).Scan(&user.Id, &user.Password)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return user, errors.New("user not found")
		}
		return user, errors.New("failed to query into database")
		//repository.Log.Panic("failed to query database", zap.Error(err))
	}

	return user, nil
}

func (repository *UserRepository) CheckUserExistence(ctx context.Context, userUUID string) error {
	query := "SELECT username FROM users WHERE id=$1"

	var username string
	err := repository.DB.QueryRow(ctx, query, userUUID).Scan(&username)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return errors.New("user already exist")
		}
		return errors.New("failed to query into database")
		//repository.Log.Panic("failed to query database", zap.Error(err))
	}

	return nil
}

func (repository *UserRepository) GetUserInfo(ctx context.Context) {

}

func (repository *UserRepository) GetAllUserData(ctx context.Context) {

}
