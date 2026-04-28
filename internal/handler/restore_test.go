package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"navbox/internal/config"
)

type stubRestorePasswordService struct {
	calls        int
	lastPassword string
	err          error
}

func (s *stubRestorePasswordService) ResetPassword(ctx context.Context, newPassword string) error {
	s.calls++
	s.lastPassword = newPassword
	return s.err
}

func TestRestoreHandlerGetStatus(t *testing.T) {
	handler := newRestoreTestHandler(&stubRestorePasswordService{})
	engine := newRestoreTestEngine(handler)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/restore/status", nil)
	engine.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"enabled":true`)
	require.Contains(t, recorder.Body.String(), `"mode":"admin-password"`)
}

func TestRestoreHandlerResetAdminPassword(t *testing.T) {
	passwordService := &stubRestorePasswordService{}
	handler := newRestoreTestHandler(passwordService)
	engine := newRestoreTestEngine(handler)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/restore/admin-password", strings.NewReader(`{
		"restore_token": "0123456789abcdef0123456789abcdef",
		"new_password": "new-password"
	}`))
	request.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Equal(t, 1, passwordService.calls)
	require.Equal(t, "new-password", passwordService.lastPassword)
}

func TestRestoreHandlerResetAdminPasswordRejectsInvalidToken(t *testing.T) {
	passwordService := &stubRestorePasswordService{}
	handler := newRestoreTestHandler(passwordService)
	engine := newRestoreTestEngine(handler)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/restore/admin-password", strings.NewReader(`{
		"restore_token": "wrong-token",
		"new_password": "new-password"
	}`))
	request.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusForbidden, recorder.Code)
	require.Equal(t, 0, passwordService.calls)
}

func TestRestoreHandlerResetAdminPasswordRejectsExpiredToken(t *testing.T) {
	passwordService := &stubRestorePasswordService{}
	handler := newRestoreTestHandler(passwordService)
	handler.tokenExpiresAt = time.Now().Add(-time.Second)
	engine := newRestoreTestEngine(handler)

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/restore/admin-password", strings.NewReader(`{
		"restore_token": "0123456789abcdef0123456789abcdef",
		"new_password": "new-password"
	}`))
	request.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusForbidden, recorder.Code)
	require.Equal(t, 0, passwordService.calls)
}

func TestRestoreHandlerResetAdminPasswordConsumesTokenOnce(t *testing.T) {
	passwordService := &stubRestorePasswordService{}
	handler := newRestoreTestHandler(passwordService)
	engine := newRestoreTestEngine(handler)

	firstRecorder := httptest.NewRecorder()
	firstRequest := httptest.NewRequest(http.MethodPost, "/api/v1/restore/admin-password", strings.NewReader(`{
		"restore_token": "0123456789abcdef0123456789abcdef",
		"new_password": "new-password"
	}`))
	firstRequest.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(firstRecorder, firstRequest)

	secondRecorder := httptest.NewRecorder()
	secondRequest := httptest.NewRequest(http.MethodPost, "/api/v1/restore/admin-password", strings.NewReader(`{
		"restore_token": "0123456789abcdef0123456789abcdef",
		"new_password": "other-password"
	}`))
	secondRequest.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(secondRecorder, secondRequest)

	require.Equal(t, http.StatusOK, firstRecorder.Code)
	require.Equal(t, http.StatusForbidden, secondRecorder.Code)
	require.Equal(t, 1, passwordService.calls)
}

func newRestoreTestHandler(passwordService *stubRestorePasswordService) *RestoreHandler {
	return &RestoreHandler{
		service: passwordService,
		cfg: config.Config{
			Restore: config.RestoreConfig{
				Mode:  config.RestoreModeAdminPassword,
				Token: "0123456789abcdef0123456789abcdef",
			},
		},
		logger:         zerolog.Nop(),
		tokenExpiresAt: time.Now().Add(restoreTokenTTL),
	}
}

func newRestoreTestEngine(handler *RestoreHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)

	engine := gin.New()
	api := engine.Group("/api/v1")
	handler.RegisterRoutes(api)
	return engine
}
