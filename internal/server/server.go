package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"go.uber.org/fx"

	"navbox/internal/config"
)

type HTTPServerParams struct {
	fx.In

	Lifecycle fx.Lifecycle
	Config    config.Config
	Logger    zerolog.Logger
	Router    *gin.Engine
}

func NewHTTPServer(params HTTPServerParams) (*http.Server, error) {
	readHeaderTimeout, err := time.ParseDuration(params.Config.HTTP.ReadHeaderTimeout)
	if err != nil {
		return nil, fmt.Errorf("parse read header timeout: %w", err)
	}
	shutdownTimeout, err := time.ParseDuration(params.Config.HTTP.ShutdownTimeout)
	if err != nil {
		return nil, fmt.Errorf("parse shutdown timeout: %w", err)
	}

	server := &http.Server{
		Addr:              params.Config.HTTP.Addr,
		Handler:           params.Router,
		ReadHeaderTimeout: readHeaderTimeout,
	}

	params.Lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			go func() {
				params.Logger.Info().Str("addr", server.Addr).Msg("HTTP server starting")
				if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
					params.Logger.Error().Err(err).Msg("HTTP server stopped unexpectedly")
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			shutdownCtx, cancel := context.WithTimeout(ctx, shutdownTimeout)
			defer cancel()

			params.Logger.Info().Msg("HTTP server shutting down")
			if err := server.Shutdown(shutdownCtx); err != nil {
				return fmt.Errorf("shutdown http server: %w", err)
			}
			return nil
		},
	})

	return server, nil
}
