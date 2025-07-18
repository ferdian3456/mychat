package config

import (
	"github.com/ferdian3456/mychat/backend/chat-service/internal/delivery/http"
	"github.com/ferdian3456/mychat/backend/chat-service/internal/delivery/http/middleware"
	"github.com/ferdian3456/mychat/backend/chat-service/internal/delivery/http/route"
	"github.com/ferdian3456/mychat/backend/chat-service/internal/repository"
	"github.com/ferdian3456/mychat/backend/chat-service/internal/usecase"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/julienschmidt/httprouter"
	"github.com/knadh/koanf/v2"
	"github.com/redis/go-redis/v9"
	zapLog "go.uber.org/zap"
)

type ServerConfig struct {
	Router  *httprouter.Router
	DB      *pgxpool.Pool
	DBCache *redis.ClusterClient
	Log     *zapLog.Logger
	Config  *koanf.Koanf
}

func Server(config *ServerConfig) {
	chatRepository := repository.NewChatRepository(config.Log, config.DB, config.DBCache)
	chatUsecase := usecase.NewChatUsecase(chatRepository, config.DB, config.Log, config.Config)
	chatController := http.NewChatController(chatUsecase, config.Log, config.Config)

	authMiddleware := middleware.NewAuthMiddleware(config.Router, config.Log, config.Config, chatUsecase)

	routeConfig := route.RouteConfig{
		Router:         config.Router,
		ChatController: chatController,
		AuthMiddleware: authMiddleware,
	}

	routeConfig.SetupRoute()
}
