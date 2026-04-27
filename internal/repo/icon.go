package repo

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"navbox/internal/model"
)

type IconRepo interface {
	Count(ctx context.Context) (int64, error)
}

type iconRepo struct {
	db *gorm.DB
}

func NewIconRepo(db *gorm.DB) IconRepo {
	return &iconRepo{db: db}
}

func (r *iconRepo) Count(ctx context.Context) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&model.Icon{}).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("count icons: %w", err)
	}
	return count, nil
}
