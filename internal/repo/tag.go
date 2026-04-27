package repo

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"navbox/internal/model"
)

type TagWithSiteCount struct {
	Tag       model.Tag
	SiteCount int64
}

type TagOrderUpdate struct {
	ID        uuid.UUID
	SortOrder int
}

type TagRepo interface {
	Count(ctx context.Context) (int64, error)
	List(ctx context.Context) ([]TagWithSiteCount, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.Tag, error)
	CountByIDs(ctx context.Context, ids []uuid.UUID) (int64, error)
	GetDefault(ctx context.Context) (*model.Tag, error)
	Create(ctx context.Context, tag *model.Tag) error
	Update(ctx context.Context, tag *model.Tag) error
	Delete(ctx context.Context, id uuid.UUID) error
	SetDefault(ctx context.Context, id uuid.UUID) error
	UpdateOrder(ctx context.Context, items []TagOrderUpdate) error
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

func (r *tagRepo) List(ctx context.Context) ([]TagWithSiteCount, error) {
	type row struct {
		ID        uuid.UUID
		Name      string
		Icon      string
		Color     string
		SortOrder int
		IsDefault bool
		IsEnabled bool
		CreatedAt time.Time
		UpdatedAt time.Time
		SiteCount int64
	}

	var rows []row
	if err := r.db.WithContext(ctx).
		Table("tags").
		Select("tags.id, tags.name, tags.icon, tags.color, tags.sort_order, tags.is_default, tags.is_enabled, tags.created_at, tags.updated_at, COUNT(site_tags.site_id) AS site_count").
		Joins("LEFT JOIN site_tags ON site_tags.tag_id = tags.id").
		Group("tags.id").
		Order("tags.sort_order ASC").
		Order("tags.created_at ASC").
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("list tags: %w", err)
	}

	result := make([]TagWithSiteCount, 0, len(rows))
	for _, item := range rows {
		result = append(result, TagWithSiteCount{
			Tag: model.Tag{
				Base: model.Base{
					ID:        item.ID,
					CreatedAt: item.CreatedAt,
					UpdatedAt: item.UpdatedAt,
				},
				Name:      item.Name,
				Icon:      item.Icon,
				Color:     item.Color,
				SortOrder: item.SortOrder,
				IsDefault: item.IsDefault,
				IsEnabled: item.IsEnabled,
			},
			SiteCount: item.SiteCount,
		})
	}
	return result, nil
}

func (r *tagRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Tag, error) {
	var tag model.Tag
	if err := r.db.WithContext(ctx).Take(&tag, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("get tag: %w", err)
	}
	return &tag, nil
}

func (r *tagRepo) CountByIDs(ctx context.Context, ids []uuid.UUID) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}

	var count int64
	if err := r.db.WithContext(ctx).Model(&model.Tag{}).Where("id IN ?", ids).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("count tags by ids: %w", err)
	}
	return count, nil
}

func (r *tagRepo) GetDefault(ctx context.Context) (*model.Tag, error) {
	var tag model.Tag
	if err := r.db.WithContext(ctx).Where("is_default = ? AND is_enabled = ?", true, true).Order("sort_order ASC").Take(&tag).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("get default tag: %w", err)
	}
	return &tag, nil
}

func (r *tagRepo) Create(ctx context.Context, tag *model.Tag) error {
	if err := r.db.WithContext(ctx).Create(tag).Error; err != nil {
		return fmt.Errorf("create tag: %w", err)
	}
	return nil
}

func (r *tagRepo) Update(ctx context.Context, tag *model.Tag) error {
	result := r.db.WithContext(ctx).
		Model(&model.Tag{}).
		Where("id = ?", tag.ID).
		Updates(map[string]any{
			"name":       tag.Name,
			"icon":       tag.Icon,
			"color":      tag.Color,
			"sort_order": tag.SortOrder,
			"is_enabled": tag.IsEnabled,
		})
	if result.Error != nil {
		return fmt.Errorf("update tag: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *tagRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("tag_id = ?", id).Delete(&model.SiteTag{}).Error; err != nil {
			return fmt.Errorf("delete tag relations: %w", err)
		}
		result := tx.Delete(&model.Tag{}, "id = ?", id)
		if result.Error != nil {
			return fmt.Errorf("delete tag: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}

func (r *tagRepo) SetDefault(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&model.Tag{}).Where("is_default = ?", true).Update("is_default", false).Error; err != nil {
			return fmt.Errorf("clear default tag: %w", err)
		}
		result := tx.Model(&model.Tag{}).Where("id = ?", id).Update("is_default", true)
		if result.Error != nil {
			return fmt.Errorf("set default tag: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}

func (r *tagRepo) UpdateOrder(ctx context.Context, items []TagOrderUpdate) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, item := range items {
			result := tx.Model(&model.Tag{}).
				Where("id = ?", item.ID).
				Update("sort_order", item.SortOrder)
			if result.Error != nil {
				return fmt.Errorf("update tag order: %w", result.Error)
			}
			if result.RowsAffected == 0 {
				return gorm.ErrRecordNotFound
			}
		}
		return nil
	})
}
