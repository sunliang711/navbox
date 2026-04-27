package server

import (
	"net/http"
	"os"
	"path/filepath"
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

func NewRouter(cfg config.Config, logger zerolog.Logger, healthHandler *handler.HealthHandler, authHandler *handler.AuthHandler, iconHandler *handler.IconHandler, importExportHandler *handler.ImportExportHandler, siteHandler *handler.SiteHandler, tagHandler *handler.TagHandler, authService service.AuthService, assets web.Assets) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	router.Use(middleware.RequestLogger(logger), gin.Recovery())

	api := router.Group("/api/v1")
	healthHandler.RegisterRoutes(api)
	siteHandler.RegisterPublicRoutes(api)
	tagHandler.RegisterPublicRoutes(api)
	authHandler.RegisterRoutes(api, middleware.AdminSession(cfg, authService))

	admin := api.Group("/admin", middleware.AdminSession(cfg, authService))
	iconHandler.RegisterAdminRoutes(admin)
	importExportHandler.RegisterAdminRoutes(admin)
	siteHandler.RegisterAdminRoutes(admin)
	tagHandler.RegisterAdminRoutes(admin)

	registerUploadRoutes(router, cfg)
	registerWebRoutes(router, assets)

	return router
}

func registerUploadRoutes(router *gin.Engine, cfg config.Config) {
	router.GET("/uploads/:file", func(c *gin.Context) {
		fileName := c.Param("file")
		if fileName == "" || fileName != filepath.Base(fileName) {
			response.Error(c, http.StatusNotFound, response.CodeNotFound, "not found")
			return
		}

		path := filepath.Join(cfg.Upload.Dir, fileName)
		info, err := os.Stat(path)
		if err != nil || info.IsDir() {
			response.Error(c, http.StatusNotFound, response.CodeNotFound, "not found")
			return
		}
		c.File(path)
	})
}

func registerWebRoutes(router *gin.Engine, assets web.Assets) {
	fileServer := http.FileServer(assets.FS)
	serveIndex := func(c *gin.Context) {
		c.Request.URL.Path = "/"
		fileServer.ServeHTTP(c.Writer, c.Request)
	}

	router.GET("/", serveIndex)
	router.GET("/assets/*filepath", func(c *gin.Context) {
		fileServer.ServeHTTP(c.Writer, c.Request)
	})
	router.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			response.Error(c, http.StatusNotFound, response.CodeNotFound, "not found")
			return
		}
		serveIndex(c)
	})
}
