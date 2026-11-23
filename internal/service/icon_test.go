package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"navbox/internal/config"
	"navbox/internal/dto"
	"navbox/internal/model"
	"navbox/internal/repo"
)

type stubIconRepo struct {
	icons []model.Icon
	refs  []repo.IconSiteRefRow
}

func (r *stubIconRepo) Count(context.Context) (int64, error) {
	return int64(len(r.icons)), nil
}

func (r *stubIconRepo) ListAll(context.Context) ([]model.Icon, error) {
	return r.icons, nil
}

func (r *stubIconRepo) ListSiteIconRefs(context.Context) ([]repo.IconSiteRefRow, error) {
	return r.refs, nil
}

func (r *stubIconRepo) GetByID(context.Context, uuid.UUID) (*model.Icon, error) {
	return nil, nil
}

func (r *stubIconRepo) GetBySHA256(context.Context, string) (*model.Icon, error) {
	return nil, nil
}

func (r *stubIconRepo) Create(context.Context, *model.Icon) error {
	return nil
}

func TestListIconsFiltersRefsAndPagination(t *testing.T) {
	now := time.Now().UTC()
	id1 := uuid.New()
	id2 := uuid.New()
	id3 := uuid.New()
	siteA := uuid.New()
	siteB := uuid.New()

	iconRepo := &stubIconRepo{
		icons: []model.Icon{
			{
				Base:      model.Base{ID: id1, CreatedAt: now},
				FileName:  "aaa111.png",
				FilePath:  "aaa111.png",
				SHA256:    "aaa111",
				MIMEType:  "image/png",
				SizeBytes: 128,
			},
			{
				Base:      model.Base{ID: id2, CreatedAt: now.Add(-time.Minute)},
				FileName:  "bbb222.svg",
				FilePath:  "bbb222.svg",
				SHA256:    "bbb222",
				MIMEType:  "image/svg+xml",
				SizeBytes: 256,
			},
			{
				Base:      model.Base{ID: id3, CreatedAt: now.Add(-2 * time.Minute)},
				FileName:  "ccc333.webp",
				FilePath:  "ccc333.webp",
				SHA256:    "ccc333",
				MIMEType:  "image/webp",
				SizeBytes: 512,
			},
		},
		refs: []repo.IconSiteRefRow{
			{SiteID: siteA, Title: "GitLab", IconValue: "/uploads/aaa111.png"},
			{SiteID: siteB, Title: "GitLab Staging", IconValue: "/uploads/aaa111.png"},
		},
	}

	svc, err := NewIconService(config.Config{}, iconRepo, nil)
	require.NoError(t, err)

	all, err := svc.ListIcons(context.Background(), dto.IconListQuery{})
	require.NoError(t, err)
	require.Equal(t, 3, all.Total)
	require.Len(t, all.Items, 3)
	require.Equal(t, "/uploads/aaa111.png", all.Items[0].URL)
	require.Len(t, all.Items[0].UsedBy, 2)
	require.Equal(t, "GitLab", all.Items[0].UsedBy[0].Title)
	require.NotNil(t, all.Items[1].UsedBy)
	require.Empty(t, all.Items[1].UsedBy)

	bySiteTitle, err := svc.ListIcons(context.Background(), dto.IconListQuery{Q: "staging"})
	require.NoError(t, err)
	require.Equal(t, 1, bySiteTitle.Total)
	require.Equal(t, id1.String(), bySiteTitle.Items[0].ID)

	byFileName, err := svc.ListIcons(context.Background(), dto.IconListQuery{Q: "bbb222"})
	require.NoError(t, err)
	require.Equal(t, 1, byFileName.Total)
	require.Equal(t, id2.String(), byFileName.Items[0].ID)

	unusedOnly, err := svc.ListIcons(context.Background(), dto.IconListQuery{Unused: true})
	require.NoError(t, err)
	require.Equal(t, 2, unusedOnly.Total)
	require.Equal(t, id2.String(), unusedOnly.Items[0].ID)
	require.Equal(t, id3.String(), unusedOnly.Items[1].ID)

	paged, err := svc.ListIcons(context.Background(), dto.IconListQuery{Limit: 1, Offset: 1})
	require.NoError(t, err)
	require.Equal(t, 3, paged.Total)
	require.Len(t, paged.Items, 1)
	require.Equal(t, id2.String(), paged.Items[0].ID)

	pastEnd, err := svc.ListIcons(context.Background(), dto.IconListQuery{Limit: 10, Offset: 10})
	require.NoError(t, err)
	require.Equal(t, 3, pastEnd.Total)
	require.NotNil(t, pastEnd.Items)
	require.Empty(t, pastEnd.Items)
}
