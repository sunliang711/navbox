package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"navbox/internal/dto"
	"navbox/internal/service"
)

type visibilitySiteService struct {
	service.SiteService
	query dto.SiteListQuery
}

func (s *visibilitySiteService) ListSites(_ context.Context, query dto.SiteListQuery) ([]dto.SiteResp, error) {
	s.query = query
	return []dto.SiteResp{}, nil
}

func TestSiteListVisibilityCannotBeOverriddenByQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	for _, admin := range []bool{false, true} {
		svc := &visibilitySiteService{}
		h := NewSiteHandler(svc, nil)
		router := gin.New()
		h.RegisterPublicRoutes(router.Group("/api/v1"))
		h.RegisterAdminRoutes(router.Group("/api/v1/admin"))
		path := "/api/v1/sites"
		if admin {
			path = "/api/v1/admin/sites"
		}
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path+"?include_disabled=true&enabled_only=false&view=favorite&search=example", nil))
		require.Equal(t, http.StatusOK, rec.Code)
		require.Equal(t, admin, svc.query.IncludeDisabled)
		require.Equal(t, "favorite", svc.query.View)
		require.Equal(t, "example", svc.query.Search)
	}
}
