package repo

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"navbox/internal/model"
)

type AuthRepo interface {
	GetAdminSetting(ctx context.Context) (*model.AdminSetting, error)
}

type authRepo struct {
	db *gorm.DB
}

func NewAuthRepo(db *gorm.DB) AuthRepo {
	return &authRepo{db: db}
}

func (r *authRepo) GetAdminSetting(ctx context.Context) (*model.AdminSetting, error) {
	var setting model.AdminSetting
	if err := r.db.WithContext(ctx).Take(&setting, "id = ?", 1).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get admin setting: %w", err)
	}
	return &setting, nil
}
