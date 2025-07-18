package http

import (
	"github.com/ferdian3456/mychat/backend/chat-service/internal/usecase"
	"github.com/julienschmidt/httprouter"
	"github.com/knadh/koanf/v2"
	"go.uber.org/zap"
	"net/http"
)

type ChatController struct {
	ChatUsecase *usecase.ChatUsecase
	Log         *zap.Logger
	Config      *koanf.Koanf
}

func NewChatController(chatUsecase *usecase.ChatUsecase, zap *zap.Logger, koanf *koanf.Koanf) *ChatController {
	return &ChatController{
		ChatUsecase: chatUsecase,
		Log:         zap,
		Config:      koanf,
	}
}

func (controller ChatController) Register(writer http.ResponseWriter, request *http.Request, params httprouter.Params) {
	
}
