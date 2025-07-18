package repository

import (
	"context"
	"errors"
	"fmt"
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

func (repository *UserRepository) RegisterWithTx(ctx context.Context, tx pgx.Tx, user model.User, errorMap map[string]string) map[string]string {
	query := "INSERT INTO users (id,username,password,created_at,updated_at) VALUES ($1,$2,$3,$4,$5)"
	_, err := tx.Exec(ctx, query, user.Id, user.Username, user.Password, user.Created_at, user.Updated_at)
	if err != nil {
		errorMap["internal"] = "failed to query into database"
		return errorMap
		//repository.Log.Panic("failed to query into database", zap.Error(err))
	}

	return nil
}

func (repository *UserRepository) CheckUsernameUniqueWithTx(ctx context.Context, tx pgx.Tx, username string, errorMap map[string]string) error {
	return nil
}

func (repository *UserRepository) AddRefreshTokenWithTx(ctx context.Context, tx pgx.Tx, refreshtoken model.RefreshToken, errorMap map[string]string) map[string]string {
	query := "INSERT INTO refresh_tokens (user_id,hashed_refresh_token,created_at,expired_at) VALUES ($1,$2,$3,$4)"
	_, err := tx.Exec(ctx, query, refreshtoken.User_id, refreshtoken.Hashed_refresh_token, refreshtoken.Created_at, refreshtoken.Expired_at)
	if err != nil {
		errorMap["internal"] = "failed to query into database"
		return errorMap
		//repository.Log.Panic("failed to query into database", zap.Error(err))
	}

	return nil
}

func (repository *UserRepository) UpdateRefreshTokenWithTx(ctx context.Context, tx pgx.Tx, tokenStatus string, userUUID string, errorMap map[string]string) map[string]string {
	query := "UPDATE refresh_tokens SET status = $1 WHERE user_id = $2 AND created_at = (SELECT MAX(created_at) FROM refresh_tokens WHERE user_id = $2)"
	_, err := tx.Exec(ctx, query, tokenStatus, userUUID)
	if err != nil {
		errorMap["internal"] = "failed to query into database"
		return errorMap
		//repository.Log.Panic("failed to query into database", zap.Error(err))
	}

	return nil
}

func (repository *UserRepository) LoginWithTx(ctx context.Context, tx pgx.Tx, username string, errorMap map[string]string) (model.User, map[string]string) {
	query := "SELECT id,password FROM users WHERE username=$1"

	var user model.User
	err := tx.QueryRow(ctx, query, username).Scan(&user.Id, &user.Password)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			errorMap["user"] = "user not found"
			return user, errorMap
		}
		errorMap["internal"] = "failed to query into database"
		return user, errorMap
		//repository.Log.Panic("failed to query database", zap.Error(err))
	}

	return user, nil
}

func (repository *UserRepository) CheckUserExistence(ctx context.Context, userUUID string, errorMap map[string]string) map[string]string {
	query := "SELECT username FROM users WHERE id=$1"

	var username string
	err := repository.DB.QueryRow(ctx, query, userUUID).Scan(&username)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			errorMap["user"] = "user not found"
			return errorMap
		}
		errorMap["internal"] = "failed to query database"
		return errorMap
		//repository.Log.Panic("failed to query database", zap.Error(err))
	}

	return nil
}

func (repository *UserRepository) GetUserInfo(ctx context.Context, userUUID string, errorMap map[string]string) (model.UserInfoResponse, map[string]string) {
	query := "SELECT id,username FROM users WHERE id=$1"

	user := model.UserInfoResponse{}
	err := repository.DB.QueryRow(ctx, query, userUUID).Scan(&user.Id, &user.Username)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			errorMap["user"] = "user not found"
			return user, errorMap
		} else {
			fmt.Println(err.Error())
			errorMap["internal"] = "failed to query into database"
			return user, errorMap
		}
	}

	return user, nil
}

func (repository *UserRepository) GetAllUserData(ctx context.Context, userUUID string, errorMap map[string]string) ([]model.AllUserInfoResponse, map[string]string) {
	query := "SELECT id,username FROM users WHERE id!=$1"

	var users []model.AllUserInfoResponse

	rows, err := repository.DB.Query(ctx, query, userUUID)
	if err != nil {
		errorMap["internal"] = "failed to query into database"
		return users, errorMap
	}

	hasData := false

	for rows.Next() {
		var user model.AllUserInfoResponse
		err = rows.Scan(&user.Id, &user.Username)
		if err != nil {
			errorMap["internal"] = "failed to scan query result"
			return users, errorMap
		}

		hasData = true
		users = append(users, user)
	}

	if hasData == false {
		errorMap["user"] = "user not found"
		return users, errorMap
	}

	return users, nil
}
