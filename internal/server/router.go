package server

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"navbox/internal/config"
	"navbox/internal/handler"
	"navbox/internal/middleware"
	"navbox/internal/response"
	"navbox/internal/service"
	"navbox/internal/web"
)

func NewRouter(cfg config.Config, logger zerolog.Logger, healthHandler *handler.HealthHandler, authHandler *handler.AuthHandler, siteHandler *handler.SiteHandler, tagHandler *handler.TagHandler, authService service.AuthService, assets web.Assets) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	router.Use(middleware.RequestLogger(logger), gin.Recovery())

	api := router.Group("/api/v1")
	healthHandler.RegisterRoutes(api)
	siteHandler.RegisterPublicRoutes(api)
	tagHandler.RegisterPublicRoutes(api)
	authHandler.RegisterRoutes(api, middleware.AdminSession(cfg, authService))

	admin := api.Group("/admin", middleware.AdminSession(cfg, authService))
	siteHandler.RegisterAdminRoutes(admin)
	tagHandler.RegisterAdminRoutes(admin)

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
