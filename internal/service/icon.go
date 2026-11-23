package service

import (
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"net/netip"
	"strings"

	"navbox/internal/config"
	"navbox/internal/dto"
	"navbox/internal/model"
	"navbox/internal/repo"
	"navbox/internal/storage"
)

type IconService interface {
	ListIcons(ctx context.Context, query dto.IconListQuery) (*dto.IconListResp, error)
	UploadIcon(ctx context.Context, file multipart.File) (*dto.IconResp, error)
	FetchIcon(ctx context.Context, websiteURL string) (*dto.IconResp, error)
}

type iconService struct {
	repo                repo.IconRepo
	store               *storage.IconStore
	allowedPrivateCIDRs []netip.Prefix
	skipTLSVerify       bool
}

func NewIconService(cfg config.Config, repo repo.IconRepo, store *storage.IconStore) (IconService, error) {
	allowedPrivateCIDRs, err := parseAllowedPrivateCIDRs(cfg.IconFetch.AllowedPrivateCIDRs)
	if err != nil {
		return nil, err
	}
	return &iconService{
		repo:                repo,
		store:               store,
		allowedPrivateCIDRs: allowedPrivateCIDRs,
		skipTLSVerify:       cfg.IconFetch.SkipTLSVerify,
	}, nil
}

const (
	defaultIconListLimit = 60
	maxIconListLimit     = 200
)

func (s *iconService) ListIcons(ctx context.Context, query dto.IconListQuery) (*dto.IconListResp, error) {
	icons, err := s.repo.ListAll(ctx)
	if err != nil {
		return nil, err
	}
	refs, err := s.repo.ListSiteIconRefs(ctx)
	if err != nil {
		return nil, err
	}

	refsByFile := make(map[string][]dto.IconSiteRef)
	for _, ref := range refs {
		fileName, ok := localIconFileName(ref.IconValue)
		if !ok {
			continue
		}
		refsByFile[fileName] = append(refsByFile[fileName], dto.IconSiteRef{
			ID:    ref.SiteID.String(),
			Title: ref.Title,
		})
	}

	q := strings.ToLower(strings.TrimSpace(query.Q))
	filtered := make([]dto.IconListItem, 0, len(icons))
	for _, icon := range icons {
		usedBy := refsByFile[icon.FileName]
		if usedBy == nil {
			usedBy = []dto.IconSiteRef{}
		}
		if query.Unused && len(usedBy) > 0 {
			continue
		}
		if q != "" && !matchesIconSearch(icon, usedBy, q) {
			continue
		}
		filtered = append(filtered, dto.IconListItem{
			IconResp: mapIcon(icon),
			UsedBy:   usedBy,
		})
	}

	limit := query.Limit
	if limit <= 0 {
		limit = defaultIconListLimit
	} else if limit > maxIconListLimit {
		limit = maxIconListLimit
	}
	offset := query.Offset
	if offset < 0 {
		offset = 0
	}

	total := len(filtered)
	if offset >= total {
		return &dto.IconListResp{
			Items: []dto.IconListItem{},
			Total: total,
		}, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return &dto.IconListResp{
		Items: filtered[offset:end],
		Total: total,
	}, nil
}

func matchesIconSearch(icon model.Icon, usedBy []dto.IconSiteRef, q string) bool {
	if strings.Contains(strings.ToLower(icon.FileName), q) {
		return true
	}
	for _, site := range usedBy {
		if strings.Contains(strings.ToLower(site.Title), q) {
			return true
		}
	}
	return false
}

func (s *iconService) UploadIcon(ctx context.Context, file multipart.File) (*dto.IconResp, error) {
	stored, err := s.store.Save(file)
	if err != nil {
		if errors.Is(err, storage.ErrInvalidIcon) || errors.Is(err, storage.ErrIconTooLarge) {
			return nil, ErrInvalidInput
		}
		return nil, err
	}
	return s.saveStoredIcon(ctx, stored)
}

func (s *iconService) saveStoredIcon(ctx context.Context, stored storage.StoredIcon) (*dto.IconResp, error) {
	existing, err := s.repo.GetBySHA256(ctx, stored.SHA256)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		resp := mapIcon(*existing)
		return &resp, nil
	}

	icon := &model.Icon{
		FileName:  stored.FileName,
		FilePath:  stored.FilePath,
		SHA256:    stored.SHA256,
		MIMEType:  stored.MIMEType,
		SizeBytes: stored.SizeBytes,
	}
	if err := s.repo.Create(ctx, icon); err != nil {
		return nil, err
	}

	resp := mapIcon(*icon)
	return &resp, nil
}

func mapIcon(icon model.Icon) dto.IconResp {
	return dto.IconResp{
		ID:        icon.ID.String(),
		FileName:  icon.FileName,
		FilePath:  icon.FilePath,
		URL:       "/uploads/" + icon.FileName,
		SHA256:    icon.SHA256,
		MIMEType:  icon.MIMEType,
		SizeBytes: icon.SizeBytes,
		CreatedAt: icon.CreatedAt,
	}
}

func parseAllowedPrivateCIDRs(value string) ([]netip.Prefix, error) {
	items := strings.Split(value, ",")
	result := make([]netip.Prefix, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		prefix, err := netip.ParsePrefix(item)
		if err != nil {
			return nil, fmt.Errorf("invalid allowed private cidr %q: %w", item, err)
		}
		result = append(result, prefix)
	}
	return result, nil
}
