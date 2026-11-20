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
	"navbox/internal/storage"
	"navbox/internal/web"
)

func Run() {
	fx.New(
		fx.Provide(
			config.Load,
			database.NewDB,
			logging.NewLogger,
			repo.NewAuthRepo,
			repo.NewAuditRepo,
			repo.NewCronRepo,
			repo.NewIconRepo,
			repo.NewImportExportRepo,
			repo.NewSiteRepo,
			repo.NewSystemSettingRepo,
			repo.NewTagRepo,
			storage.NewIconStoreFromConfig,
			service.NewAuditService,
			service.NewAuthService,
			service.NewSecretCipher,
			service.NewIconService,
			service.NewImportExportService,
			service.NewCronService,
			service.NewSiteService,
			service.NewTagService,
			handler.NewAuditHandler,
			handler.NewAuthHandler,
			handler.NewCronHandler,
			handler.NewHealthHandler,
			handler.NewIconHandler,
			handler.NewImportExportHandler,
			handler.NewRestoreHandler,
			handler.NewSiteHandler,
			handler.NewTagHandler,
			web.NewAssets,
			server.NewRouter,
			server.NewHTTPServer,
		),
		fx.Invoke(func(*gorm.DB, service.SecretCipher, service.CronService, *http.Server) {}),
	).Run()
}
