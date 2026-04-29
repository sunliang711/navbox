package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"navbox/internal/config"
	"navbox/internal/service"
)

type fakeSessionAuthService struct {
	result service.SessionValidationResult
	err    error
	token  string
}

func (s *fakeSessionAuthService) EnsureInitialized(ctx context.Context) error {
	return nil
}

func (s *fakeSessionAuthService) Login(ctx context.Context, password string) (service.LoginResult, error) {
	return service.LoginResult{}, nil
}

func (s *fakeSessionAuthService) ValidateSession(ctx context.Context, token string) error {
	s.token = token
	return s.err
}

func (s *fakeSessionAuthService) ValidateSessionWithRenewal(ctx context.Context, token string) (service.SessionValidationResult, error) {
	s.token = token
	return s.result, s.err
}

func (s *fakeSessionAuthService) Logout(ctx context.Context, token string) error {
	return nil
}

func (s *fakeSessionAuthService) ChangePassword(ctx context.Context, token string, currentPassword string, newPassword string) error {
	return nil
}

func (s *fakeSessionAuthService) ResetPassword(ctx context.Context, newPassword string) error {
	return nil
}

func TestAdminSessionRenewsCookie(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := config.Config{
		Auth: config.AuthConfig{
			SessionTTL:   "2h",
			CookieName:   "navbox_admin_session",
			CookieSecure: true,
		},
	}
	authService := &fakeSessionAuthService{
		result: service.SessionValidationResult{
			ExpiresAt: time.Now().UTC().Add(2 * time.Hour),
			Renewed:   true,
		},
	}
	router := gin.New()
	router.GET("/admin", AdminSession(cfg, authService), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.AddCookie(&http.Cookie{Name: cfg.Auth.CookieName, Value: "session-token"})
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusNoContent, recorder.Code)
	require.Equal(t, "session-token", authService.token)
	setCookie := recorder.Header().Get("Set-Cookie")
	require.Contains(t, setCookie, "navbox_admin_session=session-token")
	require.Contains(t, setCookie, "Max-Age=7200")
	require.Contains(t, setCookie, "HttpOnly")
	require.Contains(t, setCookie, "Secure")
	require.Contains(t, setCookie, "SameSite=Lax")
}

func TestAdminSessionDoesNotSetCookieWithoutRenewal(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cfg := config.Config{
		Auth: config.AuthConfig{
			SessionTTL: "2h",
			CookieName: "navbox_admin_session",
		},
	}
	authService := &fakeSessionAuthService{
		result: service.SessionValidationResult{
			ExpiresAt: time.Now().UTC().Add(90 * time.Minute),
			Renewed:   false,
		},
	}
	router := gin.New()
	router.GET("/admin", AdminSession(cfg, authService), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	req.AddCookie(&http.Cookie{Name: cfg.Auth.CookieName, Value: "session-token"})
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, req)

	require.Equal(t, http.StatusNoContent, recorder.Code)
	require.Empty(t, recorder.Header().Values("Set-Cookie"))
}
