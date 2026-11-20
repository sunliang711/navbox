package repo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"navbox/internal/model"
)

type SystemSettingRepo interface {
	Get(ctx context.Context, key string) (*model.SystemSetting, error)
	SetIfAbsent(ctx context.Context, key string, value string) error
}

type systemSettingRepo struct {
	db *gorm.DB
}

func NewSystemSettingRepo(db *gorm.DB) SystemSettingRepo {
	return &systemSettingRepo{db: db}
}

func (r *systemSettingRepo) Get(ctx context.Context, key string) (*model.SystemSetting, error) {
	var setting model.SystemSetting
	if err := r.db.WithContext(ctx).Take(&setting, "key = ?", key).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get system setting %q: %w", key, err)
	}
	return &setting, nil
}

func (r *systemSettingRepo) SetIfAbsent(ctx context.Context, key string, value string) error {
	setting := model.SystemSetting{
		Key:       key,
		ValueText: value,
		UpdatedAt: time.Now().UTC(),
	}
	if err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "key"}},
			DoNothing: true,
		}).
		Create(&setting).Error; err != nil {
		return fmt.Errorf("set system setting %q: %w", key, err)
	}
	return nil
}
