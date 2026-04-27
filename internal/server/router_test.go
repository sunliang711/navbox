package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"

	"navbox/internal/web"
)

func TestRegisterWebRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)

	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html>navbox</html>"), 0640))

	router := gin.New()
	registerWebRoutes(router, web.Assets{FS: http.Dir(dir)})

	tests := []struct {
		name       string
		path       string
		wantStatus int
		wantBody   string
	}{
		{name: "index", path: "/", wantStatus: http.StatusOK, wantBody: "navbox"},
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
		})
	}
}
