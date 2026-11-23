package handler

import (
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"navbox/internal/config"
	"navbox/internal/dto"
)

type stubListIconService struct {
	query dto.IconListQuery
	resp  *dto.IconListResp
}

func (s *stubListIconService) ListIcons(_ context.Context, query dto.IconListQuery) (*dto.IconListResp, error) {
	s.query = query
	return s.resp, nil
}

func (s *stubListIconService) UploadIcon(context.Context, multipart.File) (*dto.IconResp, error) {
	return nil, nil
}

func (s *stubListIconService) FetchIcon(context.Context, string) (*dto.IconResp, error) {
	return nil, nil
}

func TestListIconsHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	svc := &stubListIconService{
		resp: &dto.IconListResp{
			Items: []dto.IconListItem{
				{
					IconResp: dto.IconResp{
						ID:       "icon-1",
						FileName: "demo.png",
						URL:      "/uploads/demo.png",
					},
					UsedBy: []dto.IconSiteRef{{ID: "site-1", Title: "Demo"}},
				},
			},
			Total: 1,
		},
	}
	h := NewIconHandler(config.Config{}, svc, nil)
	router := gin.New()
	h.RegisterAdminRoutes(router.Group("/api/v1/admin"))

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/admin/icons?q=demo&unused=true&limit=24&offset=12", nil))
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "demo", svc.query.Q)
	require.True(t, svc.query.Unused)
	require.Equal(t, 24, svc.query.Limit)
	require.Equal(t, 12, svc.query.Offset)

	var body struct {
		Code int              `json:"code"`
		Data dto.IconListResp `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, 200, body.Code)
	require.Equal(t, 1, body.Data.Total)
	require.Len(t, body.Data.Items, 1)

	for _, badURL := range []string{
		"/api/v1/admin/icons?limit=abc",
		"/api/v1/admin/icons?offset=xyz",
		"/api/v1/admin/icons?unused=notbool",
	} {
		badRec := httptest.NewRecorder()
		router.ServeHTTP(badRec, httptest.NewRequest(http.MethodGet, badURL, nil))
		require.Equal(t, http.StatusBadRequest, badRec.Code)
	}
}
