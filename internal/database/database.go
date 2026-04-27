package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/rs/zerolog"
	"go.uber.org/fx"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"navbox/internal/config"
)

type Params struct {
	fx.In

	Lifecycle fx.Lifecycle
	Config    config.Config
	Logger    zerolog.Logger
}

func NewDB(params Params) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(params.Config.Database.DSN), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql database: %w", err)
	}
	if err := configurePool(sqlDB, params.Config.Database); err != nil {
		return nil, err
	}

	connectTimeout, err := time.ParseDuration(params.Config.Database.ConnectTimeout)
	if err != nil {
		return nil, fmt.Errorf("parse database connect timeout: %w", err)
	}

	params.Lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			connectCtx, cancel := context.WithTimeout(ctx, connectTimeout)
			defer cancel()

			if err := sqlDB.PingContext(connectCtx); err != nil {
				return fmt.Errorf("ping database: %w", err)
			}
			if err := Migrate(connectCtx, db); err != nil {
				return err
			}
			params.Logger.Info().Msg("Database connected")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			params.Logger.Info().Msg("Database connection closing")
			if err := sqlDB.Close(); err != nil {
				return fmt.Errorf("close database: %w", err)
			}
			return nil
		},
	})

	return db, nil
}

func configurePool(sqlDB *sql.DB, cfg config.DatabaseConfig) error {
	connMaxLifetime, err := time.ParseDuration(cfg.ConnMaxLifetime)
	if err != nil {
		return fmt.Errorf("parse database conn max lifetime: %w", err)
	}
	connMaxIdleTime, err := time.ParseDuration(cfg.ConnMaxIdleTime)
	if err != nil {
		return fmt.Errorf("parse database conn max idle time: %w", err)
	}

	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(connMaxLifetime)
	sqlDB.SetConnMaxIdleTime(connMaxIdleTime)
	return nil
}
