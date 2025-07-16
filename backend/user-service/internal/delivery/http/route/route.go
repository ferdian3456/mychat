package route

import (
	"github.com/ferdian3456/mychat/backend/user-service/internal/delivery/http"
	"github.com/ferdian3456/mychat/backend/user-service/internal/delivery/http/middleware"
	"github.com/julienschmidt/httprouter"
)

type RouteConfig struct {
	Router         *httprouter.Router
	UserController *http.UserController
	AuthMiddleware *middleware.AuthMiddleware
}

func (c *RouteConfig) SetupRoute() {
	c.Router.POST("/login", c.UserController.Login)
	c.Router.POST("/register", c.UserController.Register)
	c.Router.GET("/api/userinfo", c.AuthMiddleware.AuthMiddleware(c.UserController.GetUserInfo))
	c.Router.GET("/api/users", c.AuthMiddleware.AuthMiddleware(c.UserController.GetAllUserData))
}
