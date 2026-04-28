package repo

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"navbox/internal/model"
)

const (
	SiteViewFavorite      = "favorite"
	SiteViewUncategorized = "uncategorized"
	SiteTagMatchAll       = "all"
	SiteTagMatchAny       = "any"
)

type SiteListFilter struct {
	Search   string
	TagIDs   []uuid.UUID
	TagMatch string
	View     string
}

type SiteOrderUpdate struct {
	ID        uuid.UUID
	SortOrder int
}

type SiteRepo interface {
	Count(ctx context.Context) (int64, error)
	CountByIDs(ctx context.Context, ids []uuid.UUID) (int64, error)
	List(ctx context.Context, filter SiteListFilter) ([]model.Site, error)
	GetByID(ctx context.Context, id uuid.UUID) (*model.Site, error)
	GetTagsBySiteIDs(ctx context.Context, siteIDs []uuid.UUID) (map[uuid.UUID][]model.Tag, error)
	CountByTagIDs(ctx context.Context, tagIDs []uuid.UUID) (map[uuid.UUID]int64, error)
	CreateWithTags(ctx context.Context, site *model.Site, tagIDs []uuid.UUID) error
	UpdateWithTags(ctx context.Context, site *model.Site, tagIDs []uuid.UUID) error
	Delete(ctx context.Context, id uuid.UUID) error
	BatchDelete(ctx context.Context, ids []uuid.UUID) error
	BatchAddTags(ctx context.Context, siteIDs []uuid.UUID, tagIDs []uuid.UUID) error
	BatchRemoveTags(ctx context.Context, siteIDs []uuid.UUID, tagIDs []uuid.UUID) error
	UpdateOrder(ctx context.Context, items []SiteOrderUpdate) error
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

func (r *siteRepo) CountByIDs(ctx context.Context, ids []uuid.UUID) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}

	var count int64
	if err := r.db.WithContext(ctx).Model(&model.Site{}).Where("id IN ?", ids).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("count sites by ids: %w", err)
	}
	return count, nil
}

func (r *siteRepo) List(ctx context.Context, filter SiteListFilter) ([]model.Site, error) {
	query := r.db.WithContext(ctx).Model(&model.Site{})
	query = applySiteFilter(query, filter)

	var sites []model.Site
	if err := query.Order("sort_order ASC").Order("created_at ASC").Find(&sites).Error; err != nil {
		return nil, fmt.Errorf("list sites: %w", err)
	}
	return sites, nil
}

func (r *siteRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.Site, error) {
	var site model.Site
	if err := r.db.WithContext(ctx).Take(&site, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("get site: %w", err)
	}
	return &site, nil
}

func (r *siteRepo) GetTagsBySiteIDs(ctx context.Context, siteIDs []uuid.UUID) (map[uuid.UUID][]model.Tag, error) {
	result := make(map[uuid.UUID][]model.Tag)
	if len(siteIDs) == 0 {
		return result, nil
	}

	type row struct {
		SiteID    uuid.UUID
		ID        uuid.UUID
		Name      string
		Icon      string
		Color     string
		SortOrder int
		IsDefault bool
		IsEnabled bool
		CreatedAt time.Time
		UpdatedAt time.Time
	}

	var rows []row
	if err := r.db.WithContext(ctx).
		Table("site_tags").
		Select("site_tags.site_id, tags.id, tags.name, tags.icon, tags.color, tags.sort_order, tags.is_default, tags.is_enabled, tags.created_at, tags.updated_at").
		Joins("JOIN tags ON tags.id = site_tags.tag_id").
		Where("site_tags.site_id IN ?", siteIDs).
		Order("tags.sort_order ASC").
		Order("tags.created_at ASC").
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("get site tags: %w", err)
	}

	for _, item := range rows {
		result[item.SiteID] = append(result[item.SiteID], model.Tag{
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
		})
	}
	return result, nil
}

func (r *siteRepo) CountByTagIDs(ctx context.Context, tagIDs []uuid.UUID) (map[uuid.UUID]int64, error) {
	result := make(map[uuid.UUID]int64)
	if len(tagIDs) == 0 {
		return result, nil
	}

	type row struct {
		TagID uuid.UUID
		Count int64
	}
	var rows []row
	if err := r.db.WithContext(ctx).
		Table("site_tags").
		Select("tag_id, COUNT(*) AS count").
		Where("tag_id IN ?", tagIDs).
		Group("tag_id").
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("count sites by tags: %w", err)
	}
	for _, item := range rows {
		result[item.TagID] = item.Count
	}
	return result, nil
}

func (r *siteRepo) CreateWithTags(ctx context.Context, site *model.Site, tagIDs []uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(site).Error; err != nil {
			return fmt.Errorf("create site: %w", err)
		}
		if err := replaceSiteTags(tx, site.ID, tagIDs); err != nil {
			return err
		}
		return nil
	})
}

func (r *siteRepo) UpdateWithTags(ctx context.Context, site *model.Site, tagIDs []uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&model.Site{}).
			Where("id = ?", site.ID).
			Updates(map[string]any{
				"title":            site.Title,
				"description":      site.Description,
				"default_url":      site.DefaultURL,
				"lan_url":          site.LANURL,
				"open_method":      site.OpenMethod,
				"icon_type":        site.IconType,
				"icon_value":       site.IconValue,
				"background_color": site.BackgroundColor,
				"only_name":        site.OnlyName,
				"is_favorite":      site.IsFavorite,
				"sort_order":       site.SortOrder,
			})
		if result.Error != nil {
			return fmt.Errorf("update site: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		if err := replaceSiteTags(tx, site.ID, tagIDs); err != nil {
			return err
		}
		return nil
	})
}

func (r *siteRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("site_id = ?", id).Delete(&model.SiteTag{}).Error; err != nil {
			return fmt.Errorf("delete site tags: %w", err)
		}
		result := tx.Delete(&model.Site{}, "id = ?", id)
		if result.Error != nil {
			return fmt.Errorf("delete site: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}

func (r *siteRepo) BatchDelete(ctx context.Context, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("site_id IN ?", ids).Delete(&model.SiteTag{}).Error; err != nil {
			return fmt.Errorf("delete batch site tags: %w", err)
		}
		if err := tx.Where("id IN ?", ids).Delete(&model.Site{}).Error; err != nil {
			return fmt.Errorf("delete batch sites: %w", err)
		}
		return nil
	})
}

func (r *siteRepo) BatchAddTags(ctx context.Context, siteIDs []uuid.UUID, tagIDs []uuid.UUID) error {
	if len(siteIDs) == 0 || len(tagIDs) == 0 {
		return nil
	}

	relations := make([]model.SiteTag, 0, len(siteIDs)*len(tagIDs))
	for _, siteID := range siteIDs {
		for _, tagID := range tagIDs {
			relations = append(relations, model.SiteTag{SiteID: siteID, TagID: tagID})
		}
	}

	if err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		CreateInBatches(relations, 500).Error; err != nil {
		return fmt.Errorf("batch add site tags: %w", err)
	}
	return nil
}

func (r *siteRepo) BatchRemoveTags(ctx context.Context, siteIDs []uuid.UUID, tagIDs []uuid.UUID) error {
	if len(siteIDs) == 0 || len(tagIDs) == 0 {
		return nil
	}
	if err := r.db.WithContext(ctx).
		Where("site_id IN ? AND tag_id IN ?", siteIDs, tagIDs).
		Delete(&model.SiteTag{}).Error; err != nil {
		return fmt.Errorf("batch remove site tags: %w", err)
	}
	return nil
}

func (r *siteRepo) UpdateOrder(ctx context.Context, items []SiteOrderUpdate) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, item := range items {
			result := tx.Model(&model.Site{}).
				Where("id = ?", item.ID).
				Update("sort_order", item.SortOrder)
			if result.Error != nil {
				return fmt.Errorf("update site order: %w", result.Error)
			}
			if result.RowsAffected == 0 {
				return gorm.ErrRecordNotFound
			}
		}
		return nil
	})
}

func applySiteFilter(query *gorm.DB, filter SiteListFilter) *gorm.DB {
	if filter.Search != "" {
		like := "%" + filter.Search + "%"
		query = query.Where(
			"sites.title ILIKE ? OR sites.description ILIKE ? OR sites.default_url ILIKE ? OR sites.lan_url ILIKE ? OR EXISTS (SELECT 1 FROM site_tags st JOIN tags t ON t.id = st.tag_id WHERE st.site_id = sites.id AND t.name ILIKE ?)",
			like, like, like, like, like,
		)
	}
	if len(filter.TagIDs) > 0 {
		subQuery := query.Session(&gorm.Session{NewDB: true}).
			Table("site_tags").
			Select("site_id").
			Where("tag_id IN ?", filter.TagIDs).
			Group("site_id")
		if filter.TagMatch != SiteTagMatchAny {
			subQuery = subQuery.Having("COUNT(DISTINCT tag_id) = ?", len(filter.TagIDs))
		}
		query = query.Where("sites.id IN (?)", subQuery)
	}
	switch filter.View {
	case SiteViewFavorite:
		query = query.Where("sites.is_favorite = ?", true)
	case SiteViewUncategorized:
		query = query.Where("NOT EXISTS (SELECT 1 FROM site_tags st WHERE st.site_id = sites.id)")
	}
	return query
}

func replaceSiteTags(tx *gorm.DB, siteID uuid.UUID, tagIDs []uuid.UUID) error {
	if err := tx.Where("site_id = ?", siteID).Delete(&model.SiteTag{}).Error; err != nil {
		return fmt.Errorf("replace site tags: %w", err)
	}
	if len(tagIDs) == 0 {
		return nil
	}

	relations := make([]model.SiteTag, 0, len(tagIDs))
	for _, tagID := range tagIDs {
		relations = append(relations, model.SiteTag{SiteID: siteID, TagID: tagID})
	}
	if err := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&relations).Error; err != nil {
		return fmt.Errorf("create site tags: %w", err)
	}
	return nil
}
