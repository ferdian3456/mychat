package middleware

import (
	"context"
	"errors"
	"github.com/ferdian3456/mychat/backend/chat-service/internal/helper"
	"github.com/ferdian3456/mychat/backend/chat-service/internal/usecase"
	"github.com/golang-jwt/jwt/v5"
	"github.com/julienschmidt/httprouter"
	"github.com/knadh/koanf/v2"
	"go.uber.org/zap"
	"net/http"
)

type AuthMiddleware struct {
	Handler     http.Handler
	Log         *zap.Logger
	Config      *koanf.Koanf
	ChatUsecase *usecase.ChatUsecase
}

func NewAuthMiddleware(handler http.Handler, zap *zap.Logger, koanf *koanf.Koanf, chatUsecase *usecase.ChatUsecase) *AuthMiddleware {
	return &AuthMiddleware{
		Handler:     handler,
		Log:         zap,
		Config:      koanf,
		ChatUsecase: chatUsecase,
	}
}

func (middleware *AuthMiddleware) WebSocketAuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		errorMap := map[string]string{}

		ctx := request.Context()

		// Get token from URL query parameter
		wsToken := request.URL.Query().Get("websocket_token")
		if wsToken == "" {
			errorMap["auth"] = "no token provided in query"
			helper.WriteErrorResponse(writer, http.StatusBadRequest, errorMap)
			return
		}

		//fmt.Println("ws token", wsToken)

		userUUID, errorMap := middleware.ChatUsecase.VerifyWsToken(ctx, wsToken, errorMap)
		if errorMap != nil {
			if errorMap["internal"] != "" {
				helper.WriteErrorResponse(writer, http.StatusInternalServerError, errorMap)
				return
			} else {
				helper.WriteErrorResponse(writer, http.StatusUnauthorized, errorMap)
				return
			}
		}

		// Add user info to context
		ctx = context.WithValue(ctx, "user_uuid", userUUID)
		request = request.WithContext(ctx)

		next(writer, request)
	}
}

func (middleware *AuthMiddleware) AuthMiddleware(next httprouter.Handle) httprouter.Handle {
	return func(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
		var err error
		errorMap := map[string]string{}

		ctx := request.Context()

		cookie, err := request.Cookie("access_token")

		if err != nil {
			if err == http.ErrNoCookie {
				errorMap["auth"] = "no token provided"
				helper.WriteErrorResponse(writer, http.StatusUnauthorized, errorMap)
				return
			} else {
				errorMap["auth"] = err.Error()
				helper.WriteErrorResponse(writer, http.StatusUnauthorized, errorMap)
				return
			}
		}

		headerToken := cookie.Value

		secretKey := middleware.Config.String("SECRET_KEY_ACCESS_TOKEN")
		secretKeyByte := []byte(secretKey)

		token, err := jwt.Parse(headerToken, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, http.ErrNotSupported
			}
			return secretKeyByte, nil
		})

		if err != nil {
			if err == jwt.ErrTokenMalformed {
				err = errors.New("token is malformed")
				//middleware.Log.Debug(err.Error())
				errorMap["auth"] = err.Error()
				helper.WriteErrorResponse(writer, http.StatusUnauthorized, errorMap)
				return
			} else if err.Error() == "token has invalid claims: token is expired" {
				err = errors.New("token is expired")
				//middleware.Log.Debug(err.Error())
				errorMap["auth"] = err.Error()
				helper.WriteErrorResponse(writer, http.StatusUnauthorized, errorMap)
				return
			} else {
				err = errors.New("token is invalid")
				//middleware.Log.Debug(err.Error())
				errorMap["auth"] = err.Error()
				helper.WriteErrorResponse(writer, http.StatusUnauthorized, errorMap)
				return
			}
		}

		var userID string
		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			if val, exists := claims["id"]; exists {
				if strVal, ok := val.(string); ok {
					userID = strVal
				}
			} else {
				errorMap["auth"] = "token is invalid"
				helper.WriteErrorResponse(writer, http.StatusUnauthorized, errorMap)
				return
			}
		}

		errorMap = middleware.ChatUsecase.CheckUserExistance(request.Context(), userID, errorMap)
		if errorMap != nil {
			if errorMap["internal"] == "failed to query into database" {
				helper.WriteErrorResponse(writer, http.StatusInternalServerError, errorMap)
				return
			} else {
				helper.WriteErrorResponse(writer, http.StatusUnauthorized, errorMap)
				return
			}
		}

		//middleware.Log.Debug("User:" + userID)

		ctx = context.WithValue(ctx, "user_uuid", userID)
		request = request.WithContext(ctx)

		next(writer, request.WithContext(ctx), params)
	}
}
