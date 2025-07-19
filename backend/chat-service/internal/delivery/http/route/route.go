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
	c.Router.GET("/api/conversation/:id/messages", c.AuthMiddleware.AuthMiddleware(c.ChatController.GetMessage))
	c.Router.POST("/api/conversation", c.AuthMiddleware.AuthMiddleware(c.ChatController.CreateConversation))
	c.Router.GET("/api/conversation/:id/participant", c.AuthMiddleware.AuthMiddleware(c.ChatController.GetParticipantInfo))
	c.Router.GET("/api/ws-token", c.AuthMiddleware.AuthMiddleware(c.ChatController.GetWebSocketToken))
	c.Router.HandlerFunc("GET", "/api/ws", c.AuthMiddleware.WebSocketAuthMiddleware(c.ChatController.WebSocket))
}
