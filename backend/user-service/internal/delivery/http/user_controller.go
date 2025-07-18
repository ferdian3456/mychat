package http

import (
	"github.com/ferdian3456/mychat/backend/user-service/internal/helper"
	"github.com/ferdian3456/mychat/backend/user-service/internal/model"
	"github.com/ferdian3456/mychat/backend/user-service/internal/usecase"
	"github.com/julienschmidt/httprouter"
	"github.com/knadh/koanf/v2"
	"go.uber.org/zap"
	"net/http"
)

type UserController struct {
	UserUsecase *usecase.UserUsecase
	Log         *zap.Logger
	Config      *koanf.Koanf
}

func NewUserController(userUsecase *usecase.UserUsecase, zap *zap.Logger, koanf *koanf.Koanf) *UserController {
	return &UserController{
		UserUsecase: userUsecase,
		Log:         zap,
		Config:      koanf,
	}
}

func (controller UserController) Register(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
	ctx := request.Context()

	errorMap := map[string]string{}

	payload := model.UserRegisterRequest{}
	helper.ReadFromRequestBody(request, &payload)

	response, err := controller.UserUsecase.Register(ctx, payload, errorMap)
	if err != nil {
		if err["internal"] != "" {
			helper.WriteErrorResponse(writer, http.StatusInternalServerError, errorMap)
			return
		} else {
			helper.WriteErrorResponse(writer, http.StatusBadRequest, errorMap)
			return
		}
	}

	accessCookie := &http.Cookie{
		Name:     "access_token",
		Value:    response.Access_token,
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
		Expires:  response.Access_token_expires_in,
	}

	refreshCookie := &http.Cookie{
		Name:     "refresh_token",
		Value:    response.Refresh_token,
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
		Expires:  response.Refresh_token_expires_in,
	}

	http.SetCookie(writer, accessCookie)
	http.SetCookie(writer, refreshCookie)

	helper.WriteSuccessResponseNoData(writer)
}

func (controller UserController) Login(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
	ctx := request.Context()

	errorMap := map[string]string{}

	payload := model.UserRegisterRequest{}
	helper.ReadFromRequestBody(request, &payload)

	response, err := controller.UserUsecase.Login(ctx, payload, errorMap)
	if err != nil {
		if err["internal"] != "" {
			helper.WriteErrorResponse(writer, http.StatusInternalServerError, errorMap)
			return
		} else {
			helper.WriteErrorResponse(writer, http.StatusBadRequest, errorMap)
			return
		}
	}

	accessCookie := &http.Cookie{
		Name:     "access_token",
		Value:    response.Access_token,
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
		Expires:  response.Access_token_expires_in,
	}

	refreshCookie := &http.Cookie{
		Name:     "refresh_token",
		Value:    response.Refresh_token,
		Path:     "/",
		HttpOnly: true,
		Secure:   false,
		SameSite: http.SameSiteLaxMode,
		Expires:  response.Refresh_token_expires_in,
	}

	http.SetCookie(writer, accessCookie)
	http.SetCookie(writer, refreshCookie)

	helper.WriteSuccessResponseNoData(writer)
}

func (controller UserController) GetUserInfo(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
	ctx := request.Context()
	errorMap := map[string]string{}

	userUUID, _ := ctx.Value("user_uuid").(string)

	response, err := controller.UserUsecase.GetUserInfo(ctx, userUUID, errorMap)
	if err != nil {
		if err["internal"] != "" {
			helper.WriteErrorResponse(writer, http.StatusInternalServerError, errorMap)
			return
		} else {
			helper.WriteErrorResponse(writer, http.StatusBadRequest, errorMap)
			return
		}
	}

	helper.WriteSuccessResponse(writer, response)
}

func (controller UserController) GetAllUserData(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
	ctx := request.Context()
	errorMap := map[string]string{}

	userUUID, _ := ctx.Value("user_uuid").(string)

	response, err := controller.UserUsecase.GetAllUserData(ctx, userUUID, errorMap)
	if err != nil {
		if err["internal"] != "" {
			helper.WriteErrorResponse(writer, http.StatusInternalServerError, errorMap)
			return
		} else {
			helper.WriteErrorResponse(writer, http.StatusBadRequest, errorMap)
			return
		}
	}

	helper.WriteSuccessResponse(writer, response)
}
