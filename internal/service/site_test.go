package service

import (
	"context"
	"mime/multipart"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"navbox/internal/dto"
	"navbox/internal/model"
	"navbox/internal/repo"
)

type stubSiteRepo struct {
	filter repo.SiteListFilter
	created *model.Site
}

func (r *stubSiteRepo) Count(context.Context) (int64, error) { return 0, nil }
func (r *stubSiteRepo) CountByIDs(context.Context, []uuid.UUID) (int64, error) {
	return 0, nil
}
func (r *stubSiteRepo) List(_ context.Context, filter repo.SiteListFilter) ([]model.Site, error) {
	r.filter = filter
	return nil, nil
}
func (r *stubSiteRepo) GetByID(_ context.Context, id uuid.UUID) (*model.Site, error) {
	if r.created != nil && r.created.ID == id {
		return r.created, nil
	}
	return nil, nil
}
func (r *stubSiteRepo) GetTagsBySiteIDs(context.Context, []uuid.UUID) (map[uuid.UUID][]model.Tag, error) {
	return map[uuid.UUID][]model.Tag{}, nil
}
func (r *stubSiteRepo) CountByTagIDs(context.Context, []uuid.UUID) (map[uuid.UUID]int64, error) {
	return map[uuid.UUID]int64{}, nil
}
func (r *stubSiteRepo) CreateWithTags(_ context.Context, site *model.Site, _ []uuid.UUID) error {
	if site.ID == uuid.Nil {
		site.ID = uuid.New()
	}
	copySite := *site
	r.created = &copySite
	return nil
}
func (r *stubSiteRepo) UpdateWithTags(context.Context, *model.Site, []uuid.UUID) error { return nil }
func (r *stubSiteRepo) Delete(context.Context, uuid.UUID) error                        { return nil }
func (r *stubSiteRepo) BatchDelete(context.Context, []uuid.UUID) error                 { return nil }
func (r *stubSiteRepo) BatchAddTags(context.Context, []uuid.UUID, []uuid.UUID) error   { return nil }
func (r *stubSiteRepo) BatchRemoveTags(context.Context, []uuid.UUID, []uuid.UUID) error {
	return nil
}
func (r *stubSiteRepo) UpdateOrder(context.Context, []repo.SiteOrderUpdate) error { return nil }

type stubSiteTagRepo struct{}

func (r *stubSiteTagRepo) Count(context.Context) (int64, error)              { return 0, nil }
func (r *stubSiteTagRepo) List(context.Context) ([]repo.TagWithSiteCount, error) { return nil, nil }
func (r *stubSiteTagRepo) GetByID(context.Context, uuid.UUID) (*model.Tag, error) {
	return nil, nil
}
func (r *stubSiteTagRepo) CountByIDs(context.Context, []uuid.UUID) (int64, error) { return 0, nil }
func (r *stubSiteTagRepo) GetDefault(context.Context) (*model.Tag, error)         { return nil, nil }
func (r *stubSiteTagRepo) Create(context.Context, *model.Tag) error               { return nil }
func (r *stubSiteTagRepo) Update(context.Context, *model.Tag) error               { return nil }
func (r *stubSiteTagRepo) Delete(context.Context, uuid.UUID) error                { return nil }
func (r *stubSiteTagRepo) SetDefault(context.Context, uuid.UUID) error            { return nil }
func (r *stubSiteTagRepo) UpdateOrder(context.Context, []repo.TagOrderUpdate) error {
	return nil
}

type stubIconService struct {
	fetchFn func(ctx context.Context, websiteURL string) (*dto.IconResp, error)
	calls   int
}

func (s *stubIconService) UploadIcon(context.Context, multipart.File) (*dto.IconResp, error) {
	return nil, nil
}

func (s *stubIconService) FetchIcon(ctx context.Context, websiteURL string) (*dto.IconResp, error) {
	s.calls++
	return s.fetchFn(ctx, websiteURL)
}

func TestCreateSiteBoundsAutoFetchIconTimeout(t *testing.T) {
	prevTimeout := autoFetchSiteIconTimeout
	autoFetchSiteIconTimeout = 30 * time.Millisecond
	t.Cleanup(func() {
		autoFetchSiteIconTimeout = prevTimeout
	})

	var sawDeadline bool
	siteRepo := &stubSiteRepo{}
	iconSvc := &stubIconService{
		fetchFn: func(ctx context.Context, _ string) (*dto.IconResp, error) {
			_, sawDeadline = ctx.Deadline()
			<-ctx.Done()
			return nil, ctx.Err()
		},
	}
	svc := NewSiteService(siteRepo, &stubSiteTagRepo{}, iconSvc)

	start := time.Now()
	resp, err := svc.CreateSite(context.Background(), dto.SiteSaveReq{
		Title:      "Slow Site",
		DefaultURL: "https://slow.example.com",
	})
	elapsed := time.Since(start)

	require.NoError(t, err)
	require.True(t, sawDeadline)
	require.Equal(t, 1, iconSvc.calls)
	require.Less(t, elapsed, 500*time.Millisecond)
	require.Equal(t, "text", resp.IconType)
	require.Empty(t, resp.IconValue)
}

func TestCreateSiteUsesFetchedIconAndSkipsWhenProvided(t *testing.T) {
	siteRepo := &stubSiteRepo{}
	iconSvc := &stubIconService{
		fetchFn: func(context.Context, string) (*dto.IconResp, error) {
			return &dto.IconResp{URL: "/uploads/fetched.png"}, nil
		},
	}
	svc := NewSiteService(siteRepo, &stubSiteTagRepo{}, iconSvc)

	resp, err := svc.CreateSite(context.Background(), dto.SiteSaveReq{
		Title:      "Fast Site",
		DefaultURL: "https://fast.example.com",
	})
	require.NoError(t, err)
	require.Equal(t, 1, iconSvc.calls)
	require.Equal(t, "image", resp.IconType)
	require.Equal(t, "/uploads/fetched.png", resp.IconValue)

	respManual, err := svc.CreateSite(context.Background(), dto.SiteSaveReq{
		Title:      "Manual Icon Site",
		DefaultURL: "https://manual.example.com",
		IconType:   "image",
		IconValue:  "/uploads/custom.png",
	})
	require.NoError(t, err)
	require.Equal(t, 1, iconSvc.calls)
	require.Equal(t, "/uploads/custom.png", respManual.IconValue)
}

func TestSiteCreateEnabledStates(t *testing.T) {
	for _, state := range []*bool{nil, boolPointer(true), boolPointer(false)} {
		r := &stubSiteRepo{}
		svc := NewSiteService(r, &stubSiteTagRepo{}, nil)
		resp, err := svc.CreateSite(context.Background(), dto.SiteSaveReq{
			Title: "Test", DefaultURL: "https://example.com", IconValue: "N", IsEnabled: state,
		})
		require.NoError(t, err)
		require.Equal(t, state == nil || *state, resp.IsEnabled)
		require.Equal(t, state, r.created.IsEnabled)
	}
}

func TestSiteServiceVisibility(t *testing.T) {
	for _, admin := range []bool{false, true} {
		r := &stubSiteRepo{}
		svc := NewSiteService(r, &stubSiteTagRepo{}, nil)
		_, err := svc.ListSites(context.Background(), dto.SiteListQuery{IncludeDisabled: admin})
		require.NoError(t, err)
		require.Equal(t, !admin, r.filter.EnabledOnly)
	}
}
