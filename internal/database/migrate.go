package database

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"navbox/internal/model"
)

func Migrate(ctx context.Context, db *gorm.DB) error {
	if err := db.WithContext(ctx).Exec("CREATE EXTENSION IF NOT EXISTS pgcrypto").Error; err != nil {
		return fmt.Errorf("create pgcrypto extension: %w", err)
	}

	if err := db.WithContext(ctx).AutoMigrate(
		&model.Site{},
		&model.Tag{},
		&model.SiteTag{},
		&model.AdminSetting{},
		&model.AdminSession{},
		&model.Icon{},
	); err != nil {
		return fmt.Errorf("auto migrate database: %w", err)
	}

	return nil
}
