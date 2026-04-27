package repo

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"navbox/internal/model"
)

type IconRepo interface {
	Count(ctx context.Context) (int64, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.Icon, error)
	GetBySHA256(ctx context.Context, sha256 string) (*model.Icon, error)
	Create(ctx context.Context, icon *model.Icon) error
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

func (r *iconRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Icon, error) {
	var icon model.Icon
	if err := r.db.WithContext(ctx).Take(&icon, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get icon: %w", err)
	}
	return &icon, nil
}

func (r *iconRepo) GetBySHA256(ctx context.Context, sha256 string) (*model.Icon, error) {
	var icon model.Icon
	if err := r.db.WithContext(ctx).Take(&icon, "sha256 = ?", sha256).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get icon by sha256: %w", err)
	}
	return &icon, nil
}

func (r *iconRepo) Create(ctx context.Context, icon *model.Icon) error {
	if err := r.db.WithContext(ctx).Create(icon).Error; err != nil {
		return fmt.Errorf("create icon: %w", err)
	}
	return nil
}
