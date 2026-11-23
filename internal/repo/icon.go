package repo

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"navbox/internal/model"
)

type IconSiteRefRow struct {
	SiteID    uuid.UUID `gorm:"column:id"`
	Title     string    `gorm:"column:title"`
	IconValue string    `gorm:"column:icon_value"`
}

type IconRepo interface {
	Count(ctx context.Context) (int64, error)
	ListAll(ctx context.Context) ([]model.Icon, error)
	ListSiteIconRefs(ctx context.Context) ([]IconSiteRefRow, error)
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

func (r *iconRepo) ListAll(ctx context.Context) ([]model.Icon, error) {
	var icons []model.Icon
	if err := r.db.WithContext(ctx).
		Order("created_at DESC, id DESC").
		Find(&icons).Error; err != nil {
		return nil, fmt.Errorf("list icons: %w", err)
	}
	return icons, nil
}

func (r *iconRepo) ListSiteIconRefs(ctx context.Context) ([]IconSiteRefRow, error) {
	var rows []IconSiteRefRow
	if err := r.db.WithContext(ctx).
		Model(&model.Site{}).
		Select("id, title, icon_value").
		Where("icon_value LIKE ? OR icon_value LIKE ?", "/uploads/%", "uploads/%").
		Order("sort_order ASC, created_at ASC").
		Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("list site icon refs: %w", err)
	}
	return rows, nil
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
