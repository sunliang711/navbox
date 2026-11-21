package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"navbox/internal/config"
	"navbox/internal/dto"
	"navbox/internal/handler"
	"navbox/internal/service"
	"navbox/internal/web"
)

type stubRouterAuthService struct {
	token     string
	expiresAt time.Time
	ttl       time.Duration
}

func (s *stubRouterAuthService) EnsureInitialized(context.Context) error { return nil }

func (s *stubRouterAuthService) Login(_ context.Context, password string) (service.LoginResult, error) {
	if password != "secret-password" {
		return service.LoginResult{}, service.ErrInvalidCredentials
	}
	s.token = "valid-session-token"
	s.expiresAt = time.Now().UTC().Add(s.ttl)
	return service.LoginResult{Token: s.token, ExpiresAt: s.expiresAt}, nil
}

func (s *stubRouterAuthService) ValidateSession(_ context.Context, token string) error {
	if token == "" || token != s.token || !s.expiresAt.After(time.Now().UTC()) {
		return service.ErrUnauthenticated
	}
	return nil
}

func (s *stubRouterAuthService) ValidateSessionWithRenewal(_ context.Context, token string) (service.SessionValidationResult, error) {
	now := time.Now().UTC()
	if token == "" || token != s.token || !s.expiresAt.After(now) {
		return service.SessionValidationResult{}, service.ErrUnauthenticated
	}
	if s.expiresAt.Sub(now) < s.ttl/2 {
		s.expiresAt = now.Add(s.ttl)
		return service.SessionValidationResult{ExpiresAt: s.expiresAt, Renewed: true}, nil
	}
	return service.SessionValidationResult{ExpiresAt: s.expiresAt}, nil
}

func (s *stubRouterAuthService) Logout(context.Context, string) error              { return nil }
func (s *stubRouterAuthService) ChangePassword(context.Context, string, string, string) error {
	return nil
}
func (s *stubRouterAuthService) ResetPassword(context.Context, string) error { return nil }

type stubRouterSiteService struct{}

func (s *stubRouterSiteService) ListSites(context.Context, dto.SiteListQuery) ([]dto.SiteResp, error) {
	return []dto.SiteResp{}, nil
}
func (s *stubRouterSiteService) GetSite(context.Context, string) (*dto.SiteResp, error) {
	return nil, nil
}
func (s *stubRouterSiteService) CreateSite(context.Context, dto.SiteSaveReq) (*dto.SiteResp, error) {
	return nil, nil
}
func (s *stubRouterSiteService) UpdateSite(context.Context, string, dto.SiteSaveReq) (*dto.SiteResp, error) {
	return nil, nil
}
func (s *stubRouterSiteService) DeleteSite(context.Context, string) error             { return nil }
func (s *stubRouterSiteService) BatchDeleteSites(context.Context, dto.BatchDeleteSitesReq) error {
	return nil
}
func (s *stubRouterSiteService) BatchUpdateSiteTags(context.Context, dto.BatchSiteTagsReq) error {
	return nil
}
func (s *stubRouterSiteService) UpdateSiteOrder(context.Context, dto.UpdateOrderReq) error {
	return nil
}

type stubRouterTagService struct{}

func (s *stubRouterTagService) ListTags(context.Context) ([]dto.TagResp, error) {
	return []dto.TagResp{}, nil
}
func (s *stubRouterTagService) GetTag(context.Context, string) (*dto.TagResp, error) {
	return nil, nil
}
func (s *stubRouterTagService) GetPublicConfig(context.Context) (dto.PublicConfigResp, error) {
	return dto.PublicConfigResp{}, nil
}
func (s *stubRouterTagService) CreateTag(context.Context, dto.TagSaveReq) (*dto.TagResp, error) {
	return nil, nil
}
func (s *stubRouterTagService) UpdateTag(context.Context, string, dto.TagSaveReq) (*dto.TagResp, error) {
	return nil, nil
}
func (s *stubRouterTagService) DeleteTag(context.Context, string) error     { return nil }
func (s *stubRouterTagService) SetDefaultTag(context.Context, string) error { return nil }
func (s *stubRouterTagService) UpdateTagOrder(context.Context, dto.UpdateOrderReq) error {
	return nil
}

func TestRegisterWebRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html>navbox</html>"), 0640))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "favicon.ico"), []byte("ico"), 0640))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "favicon.svg"), []byte("<svg></svg>"), 0640))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "site.webmanifest"), []byte(`{"name":"Navbox"}`), 0640))

	router := gin.New()
	registerWebRoutes(router, web.Assets{FS: http.Dir(dir)})

	tests := []struct {
		name       string
		path       string
		wantStatus int
		wantBody   string
		wantType   string
	}{
		{name: "index", path: "/", wantStatus: http.StatusOK, wantBody: "navbox"},
		{name: "legacy favicon", path: "/favicon.ico", wantStatus: http.StatusOK, wantBody: "ico"},
		{name: "root asset", path: "/favicon.svg", wantStatus: http.StatusOK, wantBody: "svg"},
		{name: "manifest", path: "/site.webmanifest", wantStatus: http.StatusOK, wantBody: "Navbox", wantType: "application/manifest+json"},
		{name: "spa fallback", path: "/admin", wantStatus: http.StatusOK, wantBody: "navbox"},
		{name: "api not found", path: "/api/v1/missing", wantStatus: http.StatusNotFound, wantBody: "not found"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			recorder := httptest.NewRecorder()

			router.ServeHTTP(recorder, req)

			require.Equal(t, tt.wantStatus, recorder.Code)
			require.Contains(t, recorder.Body.String(), tt.wantBody)
			if tt.wantType != "" {
				require.Contains(t, recorder.Header().Get("Content-Type"), tt.wantType)
			}
			if tt.path == "/" || tt.path == "/admin" {
				require.Equal(t, "no-store, no-cache, must-revalidate", recorder.Header().Get("Cache-Control"))
				require.Equal(t, "no-cache", recorder.Header().Get("Pragma"))
			}
		})
	}
}

func TestRouterSessionPersistsAcrossVisitorToAdminNavigation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html>navbox</html>"), 0640))

	cfg := config.Config{
		Auth: config.AuthConfig{
			SessionTTL:            "168h",
			CookieName:            "navbox_admin_session",
			InitialPasswordLength: 16,
		},
	}
	authSvc := &stubRouterAuthService{ttl: 168 * time.Hour}
	router := NewRouter(
		cfg,
		zerolog.Nop(),
		handler.NewHealthHandler(),
		handler.NewAuthHandler(cfg, authSvc, nil),
		handler.NewIconHandler(cfg, nil, nil),
		handler.NewImportExportHandler(nil, nil, zerolog.Nop()),
		handler.NewRestoreHandler(cfg, zerolog.Nop(), authSvc, nil),
		handler.NewSiteHandler(&stubRouterSiteService{}, nil),
		handler.NewTagHandler(&stubRouterTagService{}, nil),
		handler.NewAuditHandler(nil, zerolog.Nop()),
		handler.NewCronHandler(nil, nil, zerolog.Nop()),
		authSvc,
		web.Assets{FS: http.Dir(dir)},
	)

	// Restore status endpoint returns disabled cleanly without 404.
	statusReq := httptest.NewRequest(http.MethodGet, "/api/v1/restore/status", nil)
	statusRec := httptest.NewRecorder()
	router.ServeHTTP(statusRec, statusReq)
	require.Equal(t, http.StatusOK, statusRec.Code)
	require.Contains(t, statusRec.Body.String(), `"enabled":false`)

	// 后台完整列表必须通过登录鉴权。
	adminSitesRec := httptest.NewRecorder()
	router.ServeHTTP(adminSitesRec, httptest.NewRequest(http.MethodGet, "/api/v1/admin/sites", nil))
	require.Equal(t, http.StatusUnauthorized, adminSitesRec.Code)

	// Admin logs in.
	loginReq := httptest.NewRequest(http.MethodPost, "/api/v1/admin/login", strings.NewReader(`{"password":"secret-password"}`))
	loginReq.Header.Set("Content-Type", "application/json")
	loginRec := httptest.NewRecorder()
	router.ServeHTTP(loginRec, loginReq)
	require.Equal(t, http.StatusOK, loginRec.Code)

	cookies := loginRec.Result().Cookies()
	require.Len(t, cookies, 1)
	sessionCookie := cookies[0]
	require.Equal(t, "navbox_admin_session", sessionCookie.Name)
	require.Equal(t, "valid-session-token", sessionCookie.Value)

	adminSitesReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/sites", nil)
	adminSitesReq.AddCookie(sessionCookie)
	adminSitesRec = httptest.NewRecorder()
	router.ServeHTTP(adminSitesRec, adminSitesReq)
	require.Equal(t, http.StatusOK, adminSitesRec.Code)

	// Simulate remaining TTL < 50% while browsing visitor homepage.
	authSvc.expiresAt = time.Now().UTC().Add(12 * time.Hour)

	visitorReq := httptest.NewRequest(http.MethodGet, "/api/v1/sites", nil)
	visitorReq.AddCookie(sessionCookie)
	visitorRec := httptest.NewRecorder()
	router.ServeHTTP(visitorRec, visitorReq)
	require.Equal(t, http.StatusOK, visitorRec.Code)
	renewedCookies := visitorRec.Result().Cookies()
	require.Len(t, renewedCookies, 1)
	sessionCookie = renewedCookies[0]

	// Navigate from visitor homepage to /admin and check session.
	sessionReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/session", nil)
	sessionReq.AddCookie(sessionCookie)
	sessionRec := httptest.NewRecorder()
	router.ServeHTTP(sessionRec, sessionReq)
	require.Equal(t, http.StatusOK, sessionRec.Code)
	require.Equal(t, "no-store, private", sessionRec.Header().Get("Cache-Control"))
	require.Contains(t, sessionRec.Body.String(), `"authenticated":true`)
}
