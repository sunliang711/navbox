package service

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"navbox/internal/dto"
	"navbox/internal/model"
	"navbox/internal/repo"
)

const (
	siteViewDefault       = "default"
	siteViewFavorite      = "favorite"
	siteViewUncategorized = "uncategorized"
	siteTagMatchAll       = "all"
	siteTagMatchAny       = "any"
	defaultOpenMethod     = "new_window"
	defaultIconType       = "text"
)

type SiteService interface {
	ListSites(ctx context.Context, query dto.SiteListQuery) ([]dto.SiteResp, error)
	CreateSite(ctx context.Context, req dto.SiteSaveReq) (*dto.SiteResp, error)
	UpdateSite(ctx context.Context, id string, req dto.SiteSaveReq) (*dto.SiteResp, error)
	DeleteSite(ctx context.Context, id string) error
	BatchDeleteSites(ctx context.Context, req dto.BatchDeleteSitesReq) error
	BatchUpdateSiteTags(ctx context.Context, req dto.BatchSiteTagsReq) error
	UpdateSiteOrder(ctx context.Context, req dto.UpdateOrderReq) error
}

type siteService struct {
	siteRepo    repo.SiteRepo
	tagRepo     repo.TagRepo
	iconService IconService
}

func NewSiteService(siteRepo repo.SiteRepo, tagRepo repo.TagRepo, iconService IconService) SiteService {
	return &siteService{siteRepo: siteRepo, tagRepo: tagRepo, iconService: iconService}
}

func (s *siteService) ListSites(ctx context.Context, query dto.SiteListQuery) ([]dto.SiteResp, error) {
	tagIDs, err := parseUUIDs(query.TagIDs)
	if err != nil {
		return nil, err
	}

	tagMatch := normalizeSiteTagMatch(query.TagMatch)
	if tagMatch == "" {
		return nil, ErrInvalidInput
	}

	view := strings.TrimSpace(query.View)
	if view == siteViewDefault && len(tagIDs) == 0 {
		defaultTag, err := s.tagRepo.GetDefault(ctx)
		if err != nil {
			return nil, err
		}
		if defaultTag != nil {
			tagIDs = append(tagIDs, defaultTag.ID)
		}
		view = ""
	}

	filter := repo.SiteListFilter{
		Search:   strings.TrimSpace(query.Search),
		TagIDs:   tagIDs,
		TagMatch: tagMatch,
		View:     mapSiteView(view),
	}
	sites, err := s.siteRepo.List(ctx, filter)
	if err != nil {
		return nil, err
	}
	tagsBySiteID, err := s.loadTagsBySites(ctx, sites)
	if err != nil {
		return nil, err
	}
	return mapSiteList(sites, tagsBySiteID), nil
}

func (s *siteService) CreateSite(ctx context.Context, req dto.SiteSaveReq) (*dto.SiteResp, error) {
	tagIDs, err := s.parseAndValidateTagIDs(ctx, req.TagIDs)
	if err != nil {
		return nil, err
	}

	site := siteFromReq(req)
	if shouldAutoFetchSiteIcon(site) {
		if icon, err := s.iconService.FetchIcon(ctx, site.DefaultURL); err == nil {
			site.IconType = "image"
			site.IconValue = icon.URL
		}
	}
	if err := s.siteRepo.CreateWithTags(ctx, &site, tagIDs); err != nil {
		return nil, err
	}
	return s.getSiteResp(ctx, site.ID)
}

func (s *siteService) UpdateSite(ctx context.Context, id string, req dto.SiteSaveReq) (*dto.SiteResp, error) {
	siteID, err := parseUUID(id)
	if err != nil {
		return nil, err
	}
	tagIDs, err := s.parseAndValidateTagIDs(ctx, req.TagIDs)
	if err != nil {
		return nil, err
	}

	site := siteFromReq(req)
	site.ID = siteID
	if err := s.siteRepo.UpdateWithTags(ctx, &site, tagIDs); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return s.getSiteResp(ctx, siteID)
}

func (s *siteService) DeleteSite(ctx context.Context, id string) error {
	siteID, err := parseUUID(id)
	if err != nil {
		return err
	}
	if err := s.siteRepo.Delete(ctx, siteID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

func (s *siteService) BatchDeleteSites(ctx context.Context, req dto.BatchDeleteSitesReq) error {
	siteIDs, err := parseUUIDs(req.SiteIDs)
	if err != nil {
		return err
	}
	return s.siteRepo.BatchDelete(ctx, siteIDs)
}

func (s *siteService) BatchUpdateSiteTags(ctx context.Context, req dto.BatchSiteTagsReq) error {
	siteIDs, err := parseUUIDs(req.SiteIDs)
	if err != nil {
		return err
	}
	tagIDs, err := s.parseAndValidateTagIDs(ctx, req.TagIDs)
	if err != nil {
		return err
	}
	if err := s.validateSiteIDs(ctx, siteIDs); err != nil {
		return err
	}

	switch req.Action {
	case "add":
		return s.siteRepo.BatchAddTags(ctx, siteIDs, tagIDs)
	case "remove":
		return s.siteRepo.BatchRemoveTags(ctx, siteIDs, tagIDs)
	default:
		return ErrInvalidInput
	}
}

func (s *siteService) UpdateSiteOrder(ctx context.Context, req dto.UpdateOrderReq) error {
	items := make([]repo.SiteOrderUpdate, 0, len(req.Items))
	for _, item := range req.Items {
		id, err := parseUUID(item.ID)
		if err != nil {
			return err
		}
		items = append(items, repo.SiteOrderUpdate{ID: id, SortOrder: item.SortOrder})
	}
	if err := s.siteRepo.UpdateOrder(ctx, items); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

func (s *siteService) parseAndValidateTagIDs(ctx context.Context, values []string) ([]uuid.UUID, error) {
	tagIDs, err := parseUUIDs(values)
	if err != nil {
		return nil, err
	}
	if len(tagIDs) == 0 {
		return tagIDs, nil
	}
	count, err := s.tagRepo.CountByIDs(ctx, tagIDs)
	if err != nil {
		return nil, err
	}
	if count != int64(len(tagIDs)) {
		return nil, ErrInvalidInput
	}
	return tagIDs, nil
}

func (s *siteService) validateSiteIDs(ctx context.Context, siteIDs []uuid.UUID) error {
	count, err := s.siteRepo.CountByIDs(ctx, siteIDs)
	if err != nil {
		return err
	}
	if count != int64(len(siteIDs)) {
		return ErrInvalidInput
	}
	return nil
}

func (s *siteService) getSiteResp(ctx context.Context, id uuid.UUID) (*dto.SiteResp, error) {
	site, err := s.siteRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if site == nil {
		return nil, ErrNotFound
	}
	tagsBySiteID, err := s.siteRepo.GetTagsBySiteIDs(ctx, []uuid.UUID{id})
	if err != nil {
		return nil, err
	}
	resp := mapSite(*site, tagsBySiteID[id])
	return &resp, nil
}

func (s *siteService) loadTagsBySites(ctx context.Context, sites []model.Site) (map[uuid.UUID][]model.Tag, error) {
	siteIDs := make([]uuid.UUID, 0, len(sites))
	for _, site := range sites {
		siteIDs = append(siteIDs, site.ID)
	}
	return s.siteRepo.GetTagsBySiteIDs(ctx, siteIDs)
}

func siteFromReq(req dto.SiteSaveReq) model.Site {
	openMethod := strings.TrimSpace(req.OpenMethod)
	if openMethod == "" {
		openMethod = defaultOpenMethod
	}
	iconType := strings.TrimSpace(req.IconType)
	if iconType == "" {
		iconType = defaultIconType
	}
	return model.Site{
		Title:           strings.TrimSpace(req.Title),
		Description:     strings.TrimSpace(req.Description),
		DefaultURL:      strings.TrimSpace(req.DefaultURL),
		LANURL:          strings.TrimSpace(req.LANURL),
		OpenMethod:      openMethod,
		IconType:        iconType,
		IconValue:       strings.TrimSpace(req.IconValue),
		BackgroundColor: strings.TrimSpace(req.BackgroundColor),
		OnlyName:        req.OnlyName,
		IsFavorite:      req.IsFavorite,
		SortOrder:       req.SortOrder,
	}
}

func shouldAutoFetchSiteIcon(site model.Site) bool {
	return strings.TrimSpace(site.IconValue) == "" && strings.TrimSpace(site.DefaultURL) != ""
}

func normalizeSiteTagMatch(tagMatch string) string {
	switch strings.TrimSpace(tagMatch) {
	case "", siteTagMatchAll:
		return repo.SiteTagMatchAll
	case siteTagMatchAny:
		return repo.SiteTagMatchAny
	default:
		return ""
	}
}

func mapSiteView(view string) string {
	switch view {
	case siteViewFavorite:
		return repo.SiteViewFavorite
	case siteViewUncategorized:
		return repo.SiteViewUncategorized
	default:
		return ""
	}
}

func mapSiteList(sites []model.Site, tagsBySiteID map[uuid.UUID][]model.Tag) []dto.SiteResp {
	result := make([]dto.SiteResp, 0, len(sites))
	for _, site := range sites {
		result = append(result, mapSite(site, tagsBySiteID[site.ID]))
	}
	return result
}

func mapSite(site model.Site, tags []model.Tag) dto.SiteResp {
	return dto.SiteResp{
		ID:              site.ID.String(),
		Title:           site.Title,
		Description:     site.Description,
		DefaultURL:      site.DefaultURL,
		LANURL:          site.LANURL,
		OpenMethod:      site.OpenMethod,
		IconType:        site.IconType,
		IconValue:       site.IconValue,
		BackgroundColor: site.BackgroundColor,
		OnlyName:        site.OnlyName,
		IsFavorite:      site.IsFavorite,
		SortOrder:       site.SortOrder,
		Tags:            mapTagList(tags),
		CreatedAt:       site.CreatedAt,
		UpdatedAt:       site.UpdatedAt,
	}
}
