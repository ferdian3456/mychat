package route

import (
	"github.com/ferdian3456/mychat/backend/chat-service/internal/delivery/http"
	"github.com/ferdian3456/mychat/backend/chat-service/internal/delivery/http/middleware"
	"github.com/julienschmidt/httprouter"
)

type RouteConfig struct {
	Router         *httprouter.Router
	ChatController *http.ChatController
	AuthMiddleware *middleware.AuthMiddleware
}

func (c *RouteConfig) SetupRoute() {

}
