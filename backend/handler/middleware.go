package handler

import (
	"context"
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/julienschmidt/httprouter"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"mychat/helper"
	"net/http"
)

func (h *Handler) WebSocketAuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		fmt.Println("got into middleware")
		errorMap := map[string]string{}

		ctx := request.Context()

		// Get token from URL query parameter
		wsToken := request.URL.Query().Get("websocket_token")
		if wsToken == "" {
			errorMap["auth"] = "no token provided in query"
			helper.WriteErrorResponse(writer, http.StatusBadRequest, errorMap)
			return
		}

		fmt.Println("ws token", wsToken)

		userUUID, err := h.Config.DBCache.Get(ctx, "ws_token:"+wsToken).Result()
		if err == redis.Nil {
			errorMap["auth"] = "invalid or expired ws token"
			helper.WriteErrorResponse(writer, http.StatusBadRequest, errorMap)
			return
		} else if err != nil {
			h.Config.Log.Panic("redis error", zap.Error(err))
		}

		if len(errorMap) > 0 {
			helper.WriteErrorResponse(writer, http.StatusBadRequest, errorMap)
			return
		}

		h.Config.DBCache.Del(ctx, "ws_token:"+wsToken)

		// Add user info to context
		ctx = context.WithValue(ctx, "user_uuid", userUUID)
		request = request.WithContext(ctx)

		next(writer, request)
	}
}

func (h *Handler) AuthMiddleware(next httprouter.Handle) httprouter.Handle {
	return func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		fmt.Println("got into middleware")
		var err error
		errorMap := map[string]string{}

		ctx := request.Context()

		cookie, err := request.Cookie("access_token")

		if err != nil {
			if err == http.ErrNoCookie {
				errorMap["auth"] = "no token provided"
			} else {
				errorMap["auth"] = err.Error()
			}
		}

		if len(errorMap) > 0 {
			helper.WriteErrorResponse(writer, http.StatusBadRequest, errorMap)
			return
		}

		headerToken := cookie.Value

		secretKey := h.Config.Config.String("SECRET_KEY_ACCESS_TOKEN")
		secretKeyByte := []byte(secretKey)

		token, err := jwt.Parse(headerToken, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, http.ErrNotSupported
			}
			return secretKeyByte, nil
		})

		if err != nil {
			if err == jwt.ErrTokenMalformed {
				errorMap["auth"] = "token is malformed"
			} else if err.Error() == "token has invalid claims: token is expired" {
				errorMap["auth"] = "token is expired"
			} else {
				errorMap["auth"] = "token is invalid"
			}
		}

		if len(errorMap) > 0 {
			helper.WriteErrorResponse(writer, http.StatusBadRequest, errorMap)
			return
		}

		var userID string
		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			if val, exists := claims["id"]; exists {
				if strVal, ok := val.(string); ok {
					userID = strVal
				}
			} else {
				errorMap["auth"] = "token is invalid"
			}
		}

		if len(errorMap) > 0 {
			helper.WriteErrorResponse(writer, http.StatusBadRequest, errorMap)
			return
		}

		query := "SELECT username FROM users WHERE id=$1"

		var username string
		err = h.Config.DB.QueryRow(ctx, query, userID).Scan(&username)

		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				errorMap["auth"] = "user not found, please register"
			} else {
				h.Config.Log.Panic("failed to query database", zap.Error(err))
			}
		}

		if len(errorMap) > 0 {
			helper.WriteErrorResponse(writer, http.StatusBadRequest, errorMap)
			return
		}

		h.Config.Log.Debug("User:" + userID)

		ctx = context.WithValue(ctx, "user_uuid", userID)
		ctx = context.WithValue(ctx, "username", username)
		request = request.WithContext(ctx)

		next(writer, request.WithContext(ctx), params)
	}
}
