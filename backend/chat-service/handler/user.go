package handler

import (
	"errors"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/julienschmidt/httprouter"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"mychat/helper"
	"mychat/model"
	"net/http"
	"time"
)

func (h *Handler) Register(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
	ctx := request.Context()

	errorMap := map[string]string{}
	var respErr error

	payload := model.UserRegisterRequest{}
	helper.ReadFromRequestBody(request, &payload)

	if payload.Username == "" {
		errorMap["username"] = "username is required to not be empty"
	} else if len(payload.Username) < 4 {
		errorMap["username"] = "username must be at least 4 characters"
	} else if len(payload.Username) > 22 {
		errorMap["username"] = "username must be at most 22 characters"
	}

	if payload.Password == "" {
		errorMap["password"] = "password is required to not be empty"
	} else if len(payload.Password) < 5 {
		errorMap["password"] = "password must be at least 5 characters"
	} else if len(payload.Password) > 20 {
		errorMap["password"] = "password must be at most 20 characters"
	}

	if len(errorMap) > 0 {
		helper.WriteErrorResponse(writer, http.StatusBadRequest, errorMap)
		return
	}

	// start transaction
	tx, err := h.Config.DB.Begin(ctx)
	if err != nil {
		respErr = errors.New("failed to start transaction")
		h.Config.Log.Panic(respErr.Error(), zap.Error(err))
	}

	defer helper.CommitOrRollback(ctx, tx, h.Config.Log)

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(payload.Password), bcrypt.DefaultCost)
	if err != nil {
		respErr = errors.New("error generating password hash")
		h.Config.Log.Panic(respErr.Error(), zap.Error(err))
	}

	query := "SELECT username FROM users WHERE username=$1 LIMIT 1"

	var existingUsername string
	err = tx.QueryRow(ctx, query, payload.Username).Scan(&existingUsername)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
		} else {
			h.Config.Log.Panic("failed to query database", zap.Error(err))
		}
	} else {
		errorMap["username"] = "username already exist"
	}

	if len(errorMap) > 0 {
		helper.WriteErrorResponse(writer, http.StatusBadRequest, errorMap)
		return
	}

	now := time.Now()

	user := model.User{
		Id:         uuid.New().String(),
		Username:   payload.Username,
		Password:   string(hashedPassword),
		Created_at: now,
		Updated_at: now,
	}

	query = "INSERT INTO users (id,username,password,created_at,updated_at) VALUES($1,$2,$3,$4,$5)"
	_, err = tx.Exec(ctx, query, user.Id, user.Username, user.Password, user.Created_at, user.Updated_at)
	if err != nil {
		h.Config.Log.Panic("failed to query into database", zap.Error(err))
	}

	secretKeyAccess := h.Config.Config.String("SECRET_KEY_ACCESS_TOKEN")
	secretKeyAccessByte := []byte(secretKeyAccess)

	expirationTime := now.Add(24 * time.Hour)
	expirationTimeInUnix := now.Add(24 * time.Hour).Unix()
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":  user.Id,
		"exp": expirationTimeInUnix,
	})

	accessTokenString, err := accessToken.SignedString(secretKeyAccessByte)
	if err != nil {
		h.Config.Log.Panic("failed to sign access token", zap.Error(err))
	}

	cookie := &http.Cookie{
		Name:     "access_token",
		Value:    accessTokenString,
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
		Expires:  expirationTime,
	}

	http.SetCookie(writer, cookie)

	helper.WriteSuccessResponseNoData(writer)
}

func (h *Handler) Login(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
	ctx := request.Context()

	errorMap := map[string]string{}
	var respErr error

	payload := model.UserLoginRequest{}
	helper.ReadFromRequestBody(request, &payload)

	if payload.Username == "" {
		errorMap["username"] = "username is required to not be empty"
	} else if len(payload.Username) < 4 {
		errorMap["username"] = "username must be at least 4 characters"
	} else if len(payload.Username) > 22 {
		errorMap["username"] = "username must be at most 22 characters"
	}

	if payload.Password == "" {
		errorMap["password"] = "password is required to not be empty"
	} else if len(payload.Password) < 5 {
		errorMap["password"] = "password must be at least 5 characters"
	} else if len(payload.Password) > 20 {
		errorMap["password"] = "password must be at most 20 characters"
	}

	if len(errorMap) > 0 {
		helper.WriteErrorResponse(writer, http.StatusBadRequest, errorMap)
		return
	}

	// start transaction
	tx, err := h.Config.DB.Begin(ctx)
	if err != nil {
		respErr = errors.New("failed to start transaction")
		h.Config.Log.Panic(respErr.Error(), zap.Error(err))
	}

	defer helper.CommitOrRollback(ctx, tx, h.Config.Log)

	query := "SELECT id,username,password FROM users WHERE username=$1"

	var userFromDB model.User
	err = tx.QueryRow(ctx, query, payload.Username).Scan(&userFromDB.Id, &userFromDB.Username, &userFromDB.Password)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			errorMap["username"] = "username not found"
		} else {
			h.Config.Log.Panic("failed to query database", zap.Error(err))
		}
	}

	if len(errorMap) > 0 {
		helper.WriteErrorResponse(writer, http.StatusBadRequest, errorMap)
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(userFromDB.Password), []byte(payload.Password))
	if err != nil {
		errorMap["password"] = "wrong username or password"
	}

	if len(errorMap) > 0 {
		helper.WriteErrorResponse(writer, http.StatusBadRequest, errorMap)
		return
	}

	secretKeyAccess := h.Config.Config.String("SECRET_KEY_ACCESS_TOKEN")
	secretKeyAccessByte := []byte(secretKeyAccess)

	now := time.Now()
	expirationTime := now.Add(24 * time.Hour)
	expirationTimeInUnix := now.Add(24 * time.Hour).Unix()
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":  userFromDB.Id,
		"exp": expirationTimeInUnix,
	})

	accessTokenString, err := accessToken.SignedString(secretKeyAccessByte)
	if err != nil {
		h.Config.Log.Panic("failed to sign access token", zap.Error(err))
	}

	cookie := &http.Cookie{
		Name:     "access_token",
		Value:    accessTokenString,
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
		Expires:  expirationTime,
	}

	http.SetCookie(writer, cookie)

	helper.WriteSuccessResponseNoData(writer)
}

func (h *Handler) GetUserInfo(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
	ctx := request.Context()

	userUUID, _ := ctx.Value("user_uuid").(string)
	username, _ := ctx.Value("username").(string)

	user := model.UserInfoResponse{
		Id:       userUUID,
		Username: username,
	}

	helper.WriteSuccessResponse(writer, user)
}

func (h *Handler) GetAllUserData(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
	ctx := request.Context()

	errorMap := map[string]string{}

	userUUID, _ := ctx.Value("user_uuid").(string)

	query := "SELECT id,username FROM users WHERE id!=$1"

	rows, err := h.Config.DB.Query(ctx, query, userUUID)
	if err != nil {
		h.Config.Log.Panic("failed to query into database", zap.Error(err))
	}

	defer rows.Close()

	var users []model.AllUserInfoResponse
	hasData := false

	for rows.Next() {
		var user model.AllUserInfoResponse
		err = rows.Scan(&user.Id, &user.Username)
		if err != nil {
			h.Config.Log.Panic("failed to scan query result", zap.Error(err))
		}
		users = append(users, user)
		hasData = true
	}

	if hasData == false {
		errorMap["user"] = "user not found"
	}

	if len(errorMap) > 0 {
		helper.WriteErrorResponse(writer, http.StatusBadRequest, errorMap)
		return
	}

	helper.WriteSuccessResponse(writer, users)
}
