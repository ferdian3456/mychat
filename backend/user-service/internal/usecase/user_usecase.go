package usecase

import (
	"context"
	"fmt"
	"github.com/ferdian3456/mychat/backend/user-service/internal/helper"
	"github.com/ferdian3456/mychat/backend/user-service/internal/model"
	"github.com/ferdian3456/mychat/backend/user-service/internal/repository"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/knadh/koanf/v2"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"time"
)

type UserUsecase struct {
	UserRepository *repository.UserRepository
	DB             *pgxpool.Pool
	Log            *zap.Logger
	Config         *koanf.Koanf
}

func NewUserUsecase(userRepository *repository.UserRepository, db *pgxpool.Pool, zap *zap.Logger, koanf *koanf.Koanf) *UserUsecase {
	return &UserUsecase{
		UserRepository: userRepository,
		DB:             db,
		Log:            zap,
		Config:         koanf,
	}
}

func (usecase *UserUsecase) Register(ctx context.Context, payload model.UserRegisterRequest, errorMap map[string]string) (model.Token, map[string]string) {
	token := model.Token{}

	if payload.Username == "" {
		errorMap["username"] = "username is required to not be empty"
		return token, errorMap
	} else if len(payload.Username) < 4 {
		errorMap["username"] = "username must be at least 4 characters"
		return token, errorMap
	} else if len(payload.Username) > 22 {
		errorMap["username"] = "username must be at most 22 characters"
		return token, errorMap
	}

	if payload.Password == "" {
		errorMap["password"] = "password is required to not be empty"
		return token, errorMap
	} else if len(payload.Password) < 5 {
		errorMap["password"] = "password must be at least 5 characters"
		return token, errorMap
	} else if len(payload.Password) > 20 {
		errorMap["password"] = "password must be at most 20 characters"
		return token, errorMap
	}

	// start transaction
	tx, err := usecase.DB.Begin(ctx)
	if err != nil {
		errorMap["internal"] = "failed to start transaction"
		return token, errorMap
	}

	err = usecase.UserRepository.CheckUsernameUniqueWithTx(ctx, tx, payload.Username, errorMap)
	if err != nil {
		_ = tx.Rollback(ctx)
		return token, errorMap
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(payload.Password), bcrypt.DefaultCost)
	if err != nil {
		errorMap["internal"] = "error generating password hash"
		return token, errorMap
	}

	now := time.Now()
	user := model.User{
		Id:         uuid.New().String(),
		Username:   payload.Username,
		Password:   string(hashedPassword),
		Created_at: now,
		Updated_at: now,
	}

	errorMap = usecase.UserRepository.RegisterWithTx(ctx, tx, user, errorMap)
	if errorMap != nil {
		_ = tx.Rollback(ctx)
		return token, errorMap
	}

	token, errorMap = usecase.generateToken(ctx, tx, user.Id, now, errorMap)
	if errorMap != nil {
		_ = tx.Rollback(ctx)
		return token, errorMap
	}

	err = tx.Commit(ctx)
	if err != nil {
		fmt.Println(err)
	}

	return token, nil
}

func (usecase *UserUsecase) generateToken(ctx context.Context, tx pgx.Tx, userID string, now time.Time, errorMap map[string]string) (model.Token, map[string]string) {
	token := model.Token{}

	secretKeyAccess := usecase.Config.String("SECRET_KEY_ACCESS_TOKEN")
	secretKeyAccessByte := []byte(secretKeyAccess)

	accessExpirationTime := now.Add(24 * time.Hour)
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":  userID,
		"exp": accessExpirationTime.Unix(),
	})

	accessTokenString, err := accessToken.SignedString(secretKeyAccessByte)
	if err != nil {
		errorMap["internal"] = "failed to sign access token"
		return token, errorMap
	}

	secretKeyRefresh := usecase.Config.String("SECRET_KEY_REFRESH_TOKEN")
	secretKeyRefreshByte := []byte(secretKeyRefresh)

	refreshExpirationTime := now.Add(30 * 24 * time.Hour)
	refreshToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":  userID,
		"exp": refreshExpirationTime.Unix(),
	})

	refreshTokenString, err := refreshToken.SignedString(secretKeyRefreshByte)
	if err != nil {
		errorMap["internal"] = "failed to sign access token"
		return token, errorMap
	}

	hashedRefreshToken := helper.GenerateSHA256Hash(refreshTokenString)

	refreshTokenToDB := model.RefreshToken{
		User_id:              userID,
		Hashed_refresh_token: hashedRefreshToken,
		Created_at:           now,
		Expired_at:           refreshExpirationTime,
	}

	errorMap = usecase.UserRepository.UpdateRefreshTokenWithTx(ctx, tx, "Revoke", userID, errorMap)
	if errorMap != nil {
		return token, errorMap
	}

	errorMap = usecase.UserRepository.AddRefreshTokenWithTx(ctx, tx, refreshTokenToDB, errorMap)
	if errorMap != nil {
		return token, errorMap
	}

	token = model.Token{
		Access_token:             accessTokenString,
		Access_token_expires_in:  accessExpirationTime,
		Refresh_token:            refreshTokenString,
		Refresh_token_expires_in: refreshExpirationTime,
	}

	return token, nil
}

func (usecase *UserUsecase) Login(ctx context.Context, payload model.UserRegisterRequest, errorMap map[string]string) (model.Token, map[string]string) {
	token := model.Token{}

	if payload.Username == "" {
		errorMap["username"] = "username is required to not be empty"
		return token, errorMap
	} else if len(payload.Username) < 4 {
		errorMap["username"] = "username must be at least 4 characters"
		return token, errorMap
	} else if len(payload.Username) > 22 {
		errorMap["username"] = "username must be at most 22 characters"
		return token, errorMap
	}

	if payload.Password == "" {
		errorMap["password"] = "password is required to not be empty"
		return token, errorMap
	} else if len(payload.Password) < 5 {
		errorMap["password"] = "password must be at least 5 characters"
		return token, errorMap
	} else if len(payload.Password) > 20 {
		errorMap["password"] = "password must be at most 20 characters"
		return token, errorMap
	}

	// start transaction
	tx, err := usecase.DB.Begin(ctx)
	if err != nil {
		errorMap["internal"] = "failed to start transaction"
		return token, errorMap
	}

	defer helper.CommitOrRollback(ctx, tx, usecase.Log)

	user, errorMap := usecase.UserRepository.LoginWithTx(ctx, tx, payload.Username, errorMap)
	if errorMap != nil {
		_ = tx.Rollback(ctx)
		return token, errorMap
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(payload.Password))
	if err != nil {
		errorMap["password"] = "wrong username or password"
		return token, errorMap
	}

	token, errorMap = usecase.generateToken(ctx, tx, user.Id, time.Now(), errorMap)
	if errorMap != nil {
		_ = tx.Rollback(ctx)
		return token, errorMap
	}

	return token, nil
}

func (usecase *UserUsecase) CheckUserExistance(ctx context.Context, userUUID string, errorMap map[string]string) map[string]string {
	err := usecase.UserRepository.CheckUserExistence(ctx, userUUID, errorMap)
	if err != nil {
		return err
	}

	return nil
}

func (usecase *UserUsecase) GetUserInfo(ctx context.Context, userUUID string, errorMap map[string]string) (model.UserInfoResponse, map[string]string) {
	user, errorMap := usecase.UserRepository.GetUserInfo(ctx, userUUID, errorMap)
	if errorMap != nil {
		return user, errorMap
	}

	return user, nil

}

func (usecase *UserUsecase) GetAllUserData(ctx context.Context, userUUID string, errorMap map[string]string) ([]model.AllUserInfoResponse, map[string]string) {
	user, errorMap := usecase.UserRepository.GetAllUserData(ctx, userUUID, errorMap)
	if errorMap != nil {
		return user, errorMap
	}

	return user, nil
}
