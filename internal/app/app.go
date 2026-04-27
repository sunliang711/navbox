package app

import (
	"net/http"

	"go.uber.org/fx"
	"gorm.io/gorm"

	"navbox/internal/config"
	"navbox/internal/database"
	"navbox/internal/handler"
	"navbox/internal/logging"
	"navbox/internal/repo"
	"navbox/internal/server"
	"navbox/internal/service"
	"navbox/internal/web"
)

func Run() {
	fx.New(
		fx.Provide(
			config.Load,
			database.NewDB,
			logging.NewLogger,
			repo.NewAuthRepo,
			repo.NewIconRepo,
			repo.NewSiteRepo,
			repo.NewTagRepo,
			service.NewAuthService,
			service.NewSiteService,
			service.NewTagService,
			handler.NewAuthHandler,
			handler.NewHealthHandler,
			handler.NewSiteHandler,
			handler.NewTagHandler,
			web.NewAssets,
			server.NewRouter,
			server.NewHTTPServer,
		),
		fx.Invoke(func(*gorm.DB, *http.Server) {}),
	).Run()
}
