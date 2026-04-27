package app

import (
	"net/http"

	"go.uber.org/fx"

	"navbox/internal/config"
	"navbox/internal/handler"
	"navbox/internal/logging"
	"navbox/internal/server"
	"navbox/internal/web"
)

func Run() {
	fx.New(
		fx.Provide(
			config.Load,
			logging.NewLogger,
			handler.NewHealthHandler,
			web.NewAssets,
			server.NewRouter,
			server.NewHTTPServer,
		),
		fx.Invoke(func(*http.Server) {}),
	).Run()
}
