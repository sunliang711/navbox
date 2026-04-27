package repo

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"navbox/internal/model"
)

type SiteRepo interface {
	Count(ctx context.Context) (int64, error)
}

type siteRepo struct {
	db *gorm.DB
}

func NewSiteRepo(db *gorm.DB) SiteRepo {
	return &siteRepo{db: db}
}

func (r *siteRepo) Count(ctx context.Context) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&model.Site{}).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("count sites: %w", err)
	}
	return count, nil
}
