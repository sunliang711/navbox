package service

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"

	"navbox/internal/dto"
	"navbox/internal/model"
	"navbox/internal/repo"
)

type TagService interface {
	ListTags(ctx context.Context) ([]dto.TagResp, error)
	GetPublicConfig(ctx context.Context) (dto.PublicConfigResp, error)
	CreateTag(ctx context.Context, req dto.TagSaveReq) (*dto.TagResp, error)
	UpdateTag(ctx context.Context, id string, req dto.TagSaveReq) (*dto.TagResp, error)
	DeleteTag(ctx context.Context, id string) error
	SetDefaultTag(ctx context.Context, id string) error
	UpdateTagOrder(ctx context.Context, req dto.UpdateOrderReq) error
}

type tagService struct {
	tagRepo repo.TagRepo
}

func NewTagService(tagRepo repo.TagRepo) TagService {
	return &tagService{tagRepo: tagRepo}
}

func (s *tagService) ListTags(ctx context.Context) ([]dto.TagResp, error) {
	tags, err := s.tagRepo.List(ctx)
	if err != nil {
		return nil, err
	}
	return mapTagWithSiteCountList(tags), nil
}

func (s *tagService) GetPublicConfig(ctx context.Context) (dto.PublicConfigResp, error) {
	tag, err := s.tagRepo.GetDefault(ctx)
	if err != nil {
		return dto.PublicConfigResp{}, err
	}
	if tag == nil {
		return dto.PublicConfigResp{}, nil
	}
	return dto.PublicConfigResp{DefaultTagID: tag.ID.String()}, nil
}

func (s *tagService) CreateTag(ctx context.Context, req dto.TagSaveReq) (*dto.TagResp, error) {
	tag := tagFromReq(req)
	if err := s.tagRepo.Create(ctx, &tag); err != nil {
		return nil, err
	}
	return s.getTagResp(ctx, tag.ID.String())
}

func (s *tagService) UpdateTag(ctx context.Context, id string, req dto.TagSaveReq) (*dto.TagResp, error) {
	tagID, err := parseUUID(id)
	if err != nil {
		return nil, err
	}
	tag := tagFromReq(req)
	tag.ID = tagID
	if err := s.tagRepo.Update(ctx, &tag); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return s.getTagResp(ctx, id)
}

func (s *tagService) DeleteTag(ctx context.Context, id string) error {
	tagID, err := parseUUID(id)
	if err != nil {
		return err
	}
	if err := s.tagRepo.Delete(ctx, tagID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

func (s *tagService) SetDefaultTag(ctx context.Context, id string) error {
	tagID, err := parseUUID(id)
	if err != nil {
		return err
	}
	if err := s.tagRepo.SetDefault(ctx, tagID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

func (s *tagService) UpdateTagOrder(ctx context.Context, req dto.UpdateOrderReq) error {
	items := make([]repo.TagOrderUpdate, 0, len(req.Items))
	for _, item := range req.Items {
		id, err := parseUUID(item.ID)
		if err != nil {
			return err
		}
		items = append(items, repo.TagOrderUpdate{ID: id, SortOrder: item.SortOrder})
	}
	if err := s.tagRepo.UpdateOrder(ctx, items); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		return err
	}
	return nil
}

func (s *tagService) getTagResp(ctx context.Context, id string) (*dto.TagResp, error) {
	tags, err := s.ListTags(ctx)
	if err != nil {
		return nil, err
	}
	for _, tag := range tags {
		if tag.ID == id {
			return &tag, nil
		}
	}
	return nil, ErrNotFound
}

func tagFromReq(req dto.TagSaveReq) model.Tag {
	isEnabled := true
	if req.IsEnabled != nil {
		isEnabled = *req.IsEnabled
	}
	return model.Tag{
		Name:      strings.TrimSpace(req.Name),
		Icon:      strings.TrimSpace(req.Icon),
		Color:     strings.TrimSpace(req.Color),
		SortOrder: req.SortOrder,
		IsEnabled: isEnabled,
	}
}

func mapTagWithSiteCountList(tags []repo.TagWithSiteCount) []dto.TagResp {
	result := make([]dto.TagResp, 0, len(tags))
	for _, tag := range tags {
		result = append(result, mapTagWithSiteCount(tag))
	}
	return result
}

func mapTagWithSiteCount(tag repo.TagWithSiteCount) dto.TagResp {
	resp := mapTag(tag.Tag)
	resp.SiteCount = tag.SiteCount
	return resp
}

func mapTagList(tags []model.Tag) []dto.TagResp {
	result := make([]dto.TagResp, 0, len(tags))
	for _, tag := range tags {
		result = append(result, mapTag(tag))
	}
	return result
}

func mapTag(tag model.Tag) dto.TagResp {
	return dto.TagResp{
		ID:        tag.ID.String(),
		Name:      tag.Name,
		Icon:      tag.Icon,
		Color:     tag.Color,
		SortOrder: tag.SortOrder,
		IsDefault: tag.IsDefault,
		IsEnabled: tag.IsEnabled,
		CreatedAt: tag.CreatedAt,
		UpdatedAt: tag.UpdatedAt,
	}
}
