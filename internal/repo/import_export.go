package repo

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"navbox/internal/model"
)

const importExportBatchSize = 500

type ExportFilter struct {
	SiteIDs []uuid.UUID
	TagIDs  []uuid.UUID
}

type ExportData struct {
	Sites    []model.Site
	Tags     []model.Tag
	SiteTags []model.SiteTag
}

type ImportLookupKeys struct {
	SiteIDs    []uuid.UUID
	TagIDs     []uuid.UUID
	TagNames   []string
	IconIDs    []uuid.UUID
	IconSHA256 []string
}

type ImportExistingData struct {
	Sites []model.Site
	Tags  []model.Tag
	Icons []model.Icon
}

type ImportBatch struct {
	Sites    []model.Site
	Tags     []model.Tag
	Icons    []model.Icon
	SiteTags []model.SiteTag
}

type ImportExportRepo interface {
	LoadExportData(ctx context.Context, filter ExportFilter) (ExportData, error)
	ListAllIcons(ctx context.Context) ([]model.Icon, error)
	ListIconsByFileNames(ctx context.Context, fileNames []string) ([]model.Icon, error)
	LoadImportExistingData(ctx context.Context, keys ImportLookupKeys) (ImportExistingData, error)
	Import(ctx context.Context, batch ImportBatch) error
}

type importExportRepo struct {
	db *gorm.DB
}

func NewImportExportRepo(db *gorm.DB) ImportExportRepo {
	return &importExportRepo{db: db}
}

func (r *importExportRepo) LoadExportData(ctx context.Context, filter ExportFilter) (ExportData, error) {
	if len(filter.SiteIDs) == 0 && len(filter.TagIDs) == 0 {
		return r.loadFullExportData(ctx)
	}

	siteIDs, err := r.resolveExportSiteIDs(ctx, filter)
	if err != nil {
		return ExportData{}, err
	}

	var sites []model.Site
	if len(siteIDs) > 0 {
		if err := r.db.WithContext(ctx).
			Where("id IN ?", siteIDs).
			Order("sort_order ASC").
			Order("created_at ASC").
			Find(&sites).Error; err != nil {
			return ExportData{}, fmt.Errorf("list export sites: %w", err)
		}
	}

	var siteTags []model.SiteTag
	if len(siteIDs) > 0 {
		if err := r.db.WithContext(ctx).
			Where("site_id IN ?", siteIDs).
			Order("created_at ASC").
			Find(&siteTags).Error; err != nil {
			return ExportData{}, fmt.Errorf("list export site tags: %w", err)
		}
	}

	tagIDSet := make(map[uuid.UUID]struct{}, len(filter.TagIDs)+len(siteTags))
	for _, tagID := range filter.TagIDs {
		tagIDSet[tagID] = struct{}{}
	}
	for _, relation := range siteTags {
		tagIDSet[relation.TagID] = struct{}{}
	}
	tagIDs := make([]uuid.UUID, 0, len(tagIDSet))
	for tagID := range tagIDSet {
		tagIDs = append(tagIDs, tagID)
	}

	var tags []model.Tag
	if len(tagIDs) > 0 {
		if err := r.db.WithContext(ctx).
			Where("id IN ?", tagIDs).
			Order("sort_order ASC").
			Order("created_at ASC").
			Find(&tags).Error; err != nil {
			return ExportData{}, fmt.Errorf("list export tags: %w", err)
		}
	}

	return ExportData{Sites: sites, Tags: tags, SiteTags: siteTags}, nil
}

func (r *importExportRepo) ListAllIcons(ctx context.Context) ([]model.Icon, error) {
	var icons []model.Icon
	if err := r.db.WithContext(ctx).
		Order("created_at ASC").
		Find(&icons).Error; err != nil {
		return nil, fmt.Errorf("list export icons: %w", err)
	}
	return icons, nil
}

func (r *importExportRepo) ListIconsByFileNames(ctx context.Context, fileNames []string) ([]model.Icon, error) {
	if len(fileNames) == 0 {
		return nil, nil
	}

	var icons []model.Icon
	if err := r.db.WithContext(ctx).
		Where("file_name IN ?", fileNames).
		Order("created_at ASC").
		Find(&icons).Error; err != nil {
		return nil, fmt.Errorf("list export icons by file names: %w", err)
	}
	return icons, nil
}

func (r *importExportRepo) LoadImportExistingData(ctx context.Context, keys ImportLookupKeys) (ImportExistingData, error) {
	var data ImportExistingData

	if len(keys.SiteIDs) > 0 {
		if err := r.db.WithContext(ctx).Where("id IN ?", keys.SiteIDs).Find(&data.Sites).Error; err != nil {
			return ImportExistingData{}, fmt.Errorf("list existing sites: %w", err)
		}
	}

	tagQuery := r.db.WithContext(ctx).Model(&model.Tag{})
	switch {
	case len(keys.TagIDs) > 0 && len(keys.TagNames) > 0:
		tagQuery = tagQuery.Where("id IN ? OR name IN ?", keys.TagIDs, keys.TagNames)
	case len(keys.TagIDs) > 0:
		tagQuery = tagQuery.Where("id IN ?", keys.TagIDs)
	case len(keys.TagNames) > 0:
		tagQuery = tagQuery.Where("name IN ?", keys.TagNames)
	default:
		tagQuery = nil
	}
	if tagQuery != nil {
		if err := tagQuery.Find(&data.Tags).Error; err != nil {
			return ImportExistingData{}, fmt.Errorf("list existing tags: %w", err)
		}
	}

	iconQuery := r.db.WithContext(ctx).Model(&model.Icon{})
	switch {
	case len(keys.IconIDs) > 0 && len(keys.IconSHA256) > 0:
		iconQuery = iconQuery.Where("id IN ? OR sha256 IN ?", keys.IconIDs, keys.IconSHA256)
	case len(keys.IconIDs) > 0:
		iconQuery = iconQuery.Where("id IN ?", keys.IconIDs)
	case len(keys.IconSHA256) > 0:
		iconQuery = iconQuery.Where("sha256 IN ?", keys.IconSHA256)
	default:
		iconQuery = nil
	}
	if iconQuery != nil {
		if err := iconQuery.Find(&data.Icons).Error; err != nil {
			return ImportExistingData{}, fmt.Errorf("list existing icons: %w", err)
		}
	}

	return data, nil
}

func (r *importExportRepo) Import(ctx context.Context, batch ImportBatch) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if len(batch.Tags) > 0 {
			if err := tx.CreateInBatches(batch.Tags, importExportBatchSize).Error; err != nil {
				return fmt.Errorf("import tags: %w", err)
			}
		}
		if len(batch.Icons) > 0 {
			if err := tx.CreateInBatches(batch.Icons, importExportBatchSize).Error; err != nil {
				return fmt.Errorf("import icons: %w", err)
			}
		}
		if len(batch.Sites) > 0 {
			if err := tx.CreateInBatches(batch.Sites, importExportBatchSize).Error; err != nil {
				return fmt.Errorf("import sites: %w", err)
			}
		}
		if len(batch.SiteTags) > 0 {
			if err := tx.Clauses(clause.OnConflict{DoNothing: true}).
				CreateInBatches(batch.SiteTags, importExportBatchSize).Error; err != nil {
				return fmt.Errorf("import site tags: %w", err)
			}
		}
		return nil
	})
}

func (r *importExportRepo) loadFullExportData(ctx context.Context) (ExportData, error) {
	var data ExportData
	if err := r.db.WithContext(ctx).
		Order("sort_order ASC").
		Order("created_at ASC").
		Find(&data.Sites).Error; err != nil {
		return ExportData{}, fmt.Errorf("list export sites: %w", err)
	}
	if err := r.db.WithContext(ctx).
		Order("sort_order ASC").
		Order("created_at ASC").
		Find(&data.Tags).Error; err != nil {
		return ExportData{}, fmt.Errorf("list export tags: %w", err)
	}
	if err := r.db.WithContext(ctx).
		Order("created_at ASC").
		Find(&data.SiteTags).Error; err != nil {
		return ExportData{}, fmt.Errorf("list export site tags: %w", err)
	}
	return data, nil
}

func (r *importExportRepo) resolveExportSiteIDs(ctx context.Context, filter ExportFilter) ([]uuid.UUID, error) {
	siteIDSet := make(map[uuid.UUID]struct{}, len(filter.SiteIDs))
	for _, siteID := range filter.SiteIDs {
		siteIDSet[siteID] = struct{}{}
	}

	if len(filter.TagIDs) > 0 {
		var rows []struct {
			SiteID uuid.UUID
		}
		if err := r.db.WithContext(ctx).
			Table("site_tags").
			Select("DISTINCT site_id").
			Where("tag_id IN ?", filter.TagIDs).
			Scan(&rows).Error; err != nil {
			return nil, fmt.Errorf("resolve export site ids: %w", err)
		}
		for _, row := range rows {
			siteIDSet[row.SiteID] = struct{}{}
		}
	}

	siteIDs := make([]uuid.UUID, 0, len(siteIDSet))
	for siteID := range siteIDSet {
		siteIDs = append(siteIDs, siteID)
	}
	return siteIDs, nil
}
