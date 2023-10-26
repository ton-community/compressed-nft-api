package http

import (
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/ton-community/compressed-nft-api/config"
)

func (h *Handler) RegisterHandlers(e *echo.Echo) {
	v1 := e.Group("/v1")

	v1.GET("/items", h.getItems)
	v1.GET("/items/:index", h.getItem)
	v1.GET("/state", h.getState)

	admin := e.Group("/admin")

	admin.Use(middleware.BasicAuth(func(s1, s2 string, ctx echo.Context) (bool, error) {
		return (s1 == config.Config.AdminUsername && s2 == config.Config.AdminPassword), nil
	}))

	admin.GET("/rediscover", h.rediscover)
	admin.GET("/setaddr/:addr", h.setAddr)
}
