package server

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"navbox/internal/handler"
	"navbox/internal/middleware"
	"navbox/internal/response"
	"navbox/internal/web"
)

func NewRouter(logger zerolog.Logger, healthHandler *handler.HealthHandler, assets web.Assets) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	router.Use(middleware.RequestLogger(logger), gin.Recovery())

	api := router.Group("/api/v1")
	healthHandler.RegisterRoutes(api)

	registerWebRoutes(router, assets)

	return router
}

func registerWebRoutes(router *gin.Engine, assets web.Assets) {
	fileServer := http.FileServer(assets.FS)

	router.GET("/", func(c *gin.Context) {
		c.FileFromFS("index.html", assets.FS)
	})
	router.GET("/assets/*filepath", func(c *gin.Context) {
		fileServer.ServeHTTP(c.Writer, c.Request)
	})
	router.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			response.Error(c, http.StatusNotFound, response.CodeNotFound, "not found")
			return
		}
		c.FileFromFS("index.html", assets.FS)
	})
}
