package repo

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"navbox/internal/model"
)

type TagRepo interface {
	Count(ctx context.Context) (int64, error)
}

type tagRepo struct {
	db *gorm.DB
}

func NewTagRepo(db *gorm.DB) TagRepo {
	return &tagRepo{db: db}
}

func (r *tagRepo) Count(ctx context.Context) (int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&model.Tag{}).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("count tags: %w", err)
	}
	return count, nil
}
