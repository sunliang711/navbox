package server

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"navbox/internal/config"
	"navbox/internal/database"
	"navbox/internal/dto"
	"navbox/internal/handler"
	"navbox/internal/model"
	"navbox/internal/repo"
	"navbox/internal/service"
	"navbox/internal/web"
)

type accountTestLifecycle struct{}

func (accountTestLifecycle) Append(fx.Hook) {}

type legacyAdminSession struct {
	model.Base
	TokenHash string `gorm:"type:text;not null;uniqueIndex"`
	ExpiresAt time.Time
}

func (legacyAdminSession) TableName() string { return "admin_sessions" }

func accountIntegrationDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("NAVBOX_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("NAVBOX_TEST_POSTGRES_DSN is required for isolated PostgreSQL integration tests")
	}
	base, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	schema := "account_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	require.NoError(t, base.Exec("CREATE SCHEMA "+schema).Error)
	parsed, err := pgx.ParseConfig(dsn)
	require.NoError(t, err)
	parsed.RuntimeParams["search_path"] = schema + ",public"
	sqlDB := stdlib.OpenDB(*parsed)
	db, err := gorm.Open(postgres.New(postgres.Config{Conn: sqlDB}), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, sqlDB.Close())
		require.NoError(t, base.Exec("DROP SCHEMA "+schema+" CASCADE").Error)
		baseSQL, err := base.DB()
		require.NoError(t, err)
		require.NoError(t, baseSQL.Close())
	})
	return db
}

func TestAccountsPostgresMigrationAndAPI(t *testing.T) {
	db := accountIntegrationDB(t)
	ctx := context.Background()
	// 使用旧表结构造数，验证实际的加列、账号迁移和原会话兼容。
	require.NoError(t, db.AutoMigrate(&model.AdminSetting{}, &legacyAdminSession{}))
	hash, err := bcrypt.GenerateFromPassword([]byte("legacy-password"), bcrypt.DefaultCost)
	require.NoError(t, err)
	require.NoError(t, db.Create(&model.AdminSetting{ID: 1, Initialized: true, PasswordHash: string(hash)}).Error)
	sum := sha256.Sum256([]byte("legacy-cookie"))
	require.NoError(t, db.Create(&legacyAdminSession{
		Base:      model.Base{ID: uuid.New(), CreatedAt: time.Now().Add(-time.Hour)},
		TokenHash: hex.EncodeToString(sum[:]), ExpiresAt: time.Now().Add(time.Hour),
	}).Error)
	require.NoError(t, database.Migrate(ctx, db))
	cfg := testRouterConfig()
	auth := service.NewAuthService(service.AuthServiceParams{Lifecycle: accountTestLifecycle{}, Config: cfg, Logger: zerolog.Nop(), Repo: repo.NewAuthRepo(db)})
	require.NoError(t, auth.EnsureInitialized(ctx))
	require.NoError(t, auth.EnsureInitialized(ctx))
	var accountCount int64
	require.NoError(t, db.Model(&model.AdminAccount{}).Count(&accountCount).Error)
	require.EqualValues(t, 1, accountCount)
	audit := service.NewAuditService(repo.NewAuditRepo(db), zerolog.Nop())
	router := NewRouter(cfg, zerolog.Nop(), handler.NewHealthHandler(), handler.NewAuthHandler(cfg, auth, audit),
		handler.NewIconHandler(cfg, nil, nil), handler.NewImportExportHandler(nil, nil, zerolog.Nop()),
		handler.NewRestoreHandler(cfg, zerolog.Nop(), auth, audit), handler.NewSiteHandler(&stubRouterSiteService{}, nil),
		handler.NewTagHandler(&stubRouterTagService{}, nil), handler.NewAuditHandler(nil, zerolog.Nop()),
		handler.NewCronHandler(nil, nil, zerolog.Nop()), auth, web.Assets{FS: http.Dir(t.TempDir())})
	request := func(method, path, body string, cookie *http.Cookie) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", "integration-browser")
		if cookie != nil {
			req.AddCookie(cookie)
		}
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, req)
		return recorder
	}
	adminCookie := &http.Cookie{Name: cfg.Auth.CookieName, Value: "legacy-cookie"}
	session := request(http.MethodGet, "/api/v1/admin/session", "", adminCookie)
	require.Equal(t, http.StatusOK, session.Code, session.Body.String())
	require.Contains(t, session.Body.String(), `"username":"admin"`)
	sessions := request(http.MethodGet, "/api/v1/admin/sessions", "", adminCookie)
	require.Equal(t, http.StatusOK, sessions.Code, sessions.Body.String())
	require.Contains(t, sessions.Body.String(), `"current":true`)
	require.NotContains(t, sessions.Body.String(), "token_hash")
	created := request(http.MethodPost, "/api/v1/admin/accounts", `{"username":"reader","password":"reader-password","role":"viewer","current_password":"legacy-password"}`, adminCookie)
	require.Equal(t, http.StatusOK, created.Code, created.Body.String())
	require.NotContains(t, created.Body.String(), "password")
	var createdBody struct {
		Data dto.AccountResp `json:"data"`
	}
	require.NoError(t, json.Unmarshal(created.Body.Bytes(), &createdBody))
	login := request(http.MethodPost, "/api/v1/admin/login", `{"username":"reader","password":"reader-password"}`, nil)
	require.Equal(t, http.StatusOK, login.Code, login.Body.String())
	viewerCookie := login.Result().Cookies()[0]
	for _, tt := range []struct {
		method, path, body string
		want               int
	}{
		{http.MethodGet, "/api/v1/admin/sites", "", http.StatusOK},
		{http.MethodPost, "/api/v1/admin/sites", `{}`, http.StatusForbidden},
		{http.MethodPut, "/api/v1/admin/tags/order", `{}`, http.StatusForbidden},
		{http.MethodPost, "/api/v1/admin/export", `{}`, http.StatusForbidden},
		{http.MethodGet, "/api/v1/admin/cron-jobs/example/secrets", "", http.StatusForbidden},
		{http.MethodGet, "/api/v1/admin/accounts", "", http.StatusForbidden},
		{http.MethodPost, "/api/v1/admin/accounts", `{"username":"hacker","password":"reader-password","role":"admin","current_password":"reader-password"}`, http.StatusForbidden},
		{http.MethodDelete, "/api/v1/admin/sessions/not-a-uuid", "", http.StatusBadRequest},
		{http.MethodGet, "/api/v1/admin/sessions", "", http.StatusOK},
	} {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			response := request(tt.method, tt.path, tt.body, viewerCookie)
			require.Equal(t, tt.want, response.Code, response.Body.String())
		})
	}
	denied := request(http.MethodGet, "/api/v1/admin/accounts", "", nil)
	require.Equal(t, http.StatusUnauthorized, denied.Code)
	sessions = request(http.MethodGet, "/api/v1/admin/sessions", "", viewerCookie)
	var sessionBody struct {
		Data struct {
			Sessions []dto.ActiveSessionResp `json:"sessions"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(sessions.Body.Bytes(), &sessionBody))
	require.Len(t, sessionBody.Data.Sessions, 1)
	require.Equal(t, "integration-browser", sessionBody.Data.Sessions[0].UserAgent)
	require.NotEmpty(t, sessionBody.Data.Sessions[0].ClientIP)
	revoked := request(http.MethodDelete, "/api/v1/admin/sessions/"+sessionBody.Data.Sessions[0].ID, "", viewerCookie)
	require.Equal(t, http.StatusOK, revoked.Code, revoked.Body.String())
	require.Contains(t, revoked.Body.String(), `"current":true`)
	require.Equal(t, -1, revoked.Result().Cookies()[0].MaxAge)
	require.Equal(t, http.StatusUnauthorized, request(http.MethodGet, "/api/v1/admin/session", "", viewerCookie).Code)
	disabled := request(http.MethodPut, "/api/v1/admin/accounts/"+createdBody.Data.ID, `{"current_password":"legacy-password","disabled":true}`, adminCookie)
	require.Equal(t, http.StatusOK, disabled.Code, disabled.Body.String())
	require.Equal(t, http.StatusUnauthorized, request(http.MethodPost, "/api/v1/admin/login", `{"username":"reader","password":"reader-password"}`, nil).Code)
	var events []model.AuditLog
	require.NoError(t, db.Where("action = ? AND status = ?", "admin.create", service.AuditStatusSuccess).Find(&events).Error)
	require.Len(t, events, 1)
	require.Equal(t, "admin", events[0].ActorName)
	require.NotEqual(t, "admin", events[0].ActorID)
	require.NotContains(t, events[0].DetailJSON, "legacy-password")
	require.NotContains(t, events[0].DetailJSON, "reader-password")
}

func testRouterConfig() config.Config {
	return config.Config{Auth: config.AuthConfig{SessionTTL: "2h", CookieName: "navbox_admin_session", InitialPasswordLength: 16}}
}

func TestAccountsPostgresConcurrentLastAdminProtection(t *testing.T) {
	db := accountIntegrationDB(t)
	ctx := context.Background()
	require.NoError(t, database.Migrate(ctx, db))
	r := repo.NewAuthRepo(db)
	hash, err := bcrypt.GenerateFromPassword([]byte("admin-password"), bcrypt.DefaultCost)
	require.NoError(t, err)
	require.NoError(t, r.SaveAdminSetting(ctx, &model.AdminSetting{ID: 1, Initialized: true, PasswordHash: string(hash)}))
	auth := service.NewAuthService(service.AuthServiceParams{Lifecycle: accountTestLifecycle{}, Config: testRouterConfig(), Logger: zerolog.Nop(), Repo: r})
	require.NoError(t, auth.EnsureInitialized(ctx))
	first, err := auth.Login(ctx, "admin", "admin-password", "", "")
	require.NoError(t, err)
	secondAccount, err := auth.CreateAccount(ctx, first.Token, dto.CreateAccountReq{Username: "second-admin", Password: "second-password", Role: model.RoleAdmin, CurrentPassword: "admin-password"})
	require.NoError(t, err)
	second, err := auth.Login(ctx, secondAccount.Username, "second-password", "", "")
	require.NoError(t, err)
	start := make(chan struct{})
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for _, actor := range []struct{ token, id, password string }{{first.Token, first.User.ID, "admin-password"}, {second.Token, second.User.ID, "second-password"}} {
		wg.Add(1)
		go func(token, id, password string) {
			defer wg.Done()
			<-start
			role := model.RoleViewer
			_, err := auth.UpdateAccount(ctx, token, uuid.MustParse(id), dto.UpdateAccountReq{CurrentPassword: password, Role: &role})
			results <- err
		}(actor.token, actor.id, actor.password)
	}
	close(start)
	wg.Wait()
	close(results)
	succeeded, protected := 0, 0
	for err := range results {
		if err == nil {
			succeeded++
		} else if errors.Is(err, service.ErrLastAdmin) {
			protected++
		} else {
			require.NoError(t, err)
		}
	}
	require.Equal(t, 1, succeeded)
	require.Equal(t, 1, protected)
	var count int64
	require.NoError(t, db.Model(&model.AdminAccount{}).Where("role = ? AND disabled = ?", model.RoleAdmin, false).Count(&count).Error)
	require.EqualValues(t, 1, count)
}
