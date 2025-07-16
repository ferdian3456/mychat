package config

import (
	"github.com/ferdian3456/mychat/backend/user-service/internal/delivery/http"
	"github.com/ferdian3456/mychat/backend/user-service/internal/delivery/http/middleware"
	"github.com/ferdian3456/mychat/backend/user-service/internal/delivery/http/route"
	"github.com/ferdian3456/mychat/backend/user-service/internal/repository"
	"github.com/ferdian3456/mychat/backend/user-service/internal/usecase"
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
	userRepository := repository.NewUserRepository(config.Log, config.DB, config.DBCache)
	userUsecase := usecase.NewUserUsecase(userRepository, config.DB, config.Log, config.Config)
	userController := http.NewUserController(userUsecase, config.Log, config.Config)

	authMiddleware := middleware.NewAuthMiddleware(config.Router, config.Log, config.Config, userUsecase)

	routeConfig := route.RouteConfig{
		Router:         config.Router,
		UserController: userController,
		AuthMiddleware: authMiddleware,
	}

	routeConfig.SetupRoute()
}
