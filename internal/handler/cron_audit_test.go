package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"navbox/internal/dto"
	"navbox/internal/service"
)

type stubAuditService struct {
	recorded []service.AuditEvent
	listResp dto.AuditListResp
	exports  []dto.AuditLogResp
}

func (s *stubAuditService) Record(_ context.Context, ev service.AuditEvent) {
	s.recorded = append(s.recorded, ev)
}

func (s *stubAuditService) List(_ context.Context, _ dto.AuditListQuery) (dto.AuditListResp, error) {
	return s.listResp, nil
}

func (s *stubAuditService) ExportAll(_ context.Context, _ dto.AuditListQuery) ([]dto.AuditLogResp, error) {
	return s.exports, nil
}

func (s *stubAuditService) CleanupBefore(_ context.Context, _ time.Time) (int64, error) {
	return 0, nil
}

type stubCronService struct {
	jobs    []dto.CronJobResp
	runs    []dto.CronJobRunResp
	secrets dto.CronJobSecretsResp
	testRes dto.TestSafeKeepResp
	err     error
}

func (s *stubCronService) ListJobs(_ context.Context) ([]dto.CronJobResp, error) {
	return s.jobs, s.err
}

func (s *stubCronService) UpdateJob(_ context.Context, _ string, _ dto.UpdateCronJobReq) (dto.CronJobResp, error) {
	if s.err != nil {
		return dto.CronJobResp{}, s.err
	}
	return s.jobs[0], nil
}

func (s *stubCronService) RunNow(_ context.Context, _ string, _ string) (dto.CronJobRunResp, error) {
	if s.err != nil {
		return dto.CronJobRunResp{}, s.err
	}
	return s.runs[0], nil
}

func (s *stubCronService) ListRuns(_ context.Context, _ string, _ int) ([]dto.CronJobRunResp, error) {
	return s.runs, s.err
}

func (s *stubCronService) RevealSecrets(_ context.Context, _ string) (dto.CronJobSecretsResp, error) {
	return s.secrets, s.err
}

func (s *stubCronService) TestSafeKeepConnection(_ context.Context, _ string, _ dto.TestSafeKeepReq) (dto.TestSafeKeepResp, error) {
	return s.testRes, s.err
}

func (s *stubCronService) ReloadAll(_ context.Context) error {
	return nil
}

func TestAuditHandlerListAndExport(t *testing.T) {
	gin.SetMode(gin.TestMode)
	now := time.Date(2026, 11, 20, 4, 0, 0, 0, time.UTC)
	auditSvc := &stubAuditService{
		listResp: dto.AuditListResp{
			Total: 1,
			Items: []dto.AuditLogResp{
				{
					ID:           1,
					ActorType:    "admin",
					ActorName:    "admin",
					Action:       "site.create",
					ResourceType: "site",
					ResourceID:   "site-1",
					Status:       "success",
					IPAddress:    "127.0.0.1",
					DetailJSON:   `{"title":"Demo"}`,
					CreatedAt:    now,
				},
			},
		},
		exports: []dto.AuditLogResp{
			{
				ID:           1,
				ActorType:    "admin",
				ActorName:    "admin",
				Action:       "site.create",
				ResourceType: "site",
				ResourceID:   "site-1",
				Status:       "success",
				IPAddress:    "127.0.0.1",
				DetailJSON:   `{"title":"Demo"}`,
				CreatedAt:    now,
			},
		},
	}

	router := gin.New()
	admin := router.Group("/api/v1/admin")
	NewAuditHandler(auditSvc, zerolog.Nop()).RegisterAdminRoutes(admin)

	// 1. 正常查询
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/audit-logs?actor_type=admin&limit=50", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"action":"site.create"`)

	// 2. 非法时间范围返回 400
	badRec := httptest.NewRecorder()
	badReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/audit-logs?from=2026-11-20T10:00:00Z&to=2026-11-20T09:00:00Z", nil)
	router.ServeHTTP(badRec, badReq)
	require.Equal(t, http.StatusBadRequest, badRec.Code)

	// 3. 导出 CSV
	csvRec := httptest.NewRecorder()
	csvReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/audit-logs/export?format=csv", nil)
	router.ServeHTTP(csvRec, csvReq)
	require.Equal(t, http.StatusOK, csvRec.Code)
	require.Contains(t, csvRec.Header().Get("Content-Type"), "text/csv")
	require.Contains(t, csvRec.Body.String(), "site.create")

	// 4. 导出 JSON
	jsonRec := httptest.NewRecorder()
	jsonReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/audit-logs/export?format=json", nil)
	router.ServeHTTP(jsonRec, jsonReq)
	require.Equal(t, http.StatusOK, jsonRec.Code)
	require.Contains(t, jsonRec.Header().Get("Content-Type"), "application/json")
	var exported []dto.AuditLogResp
	require.NoError(t, json.Unmarshal(jsonRec.Body.Bytes(), &exported))
	require.Len(t, exported, 1)
}

func TestCronHandlerRoutesAndAuditRecording(t *testing.T) {
	gin.SetMode(gin.TestMode)
	auditSvc := &stubAuditService{}
	cronSvc := &stubCronService{
		jobs: []dto.CronJobResp{
			{
				ID:              "11111111-1111-1111-1111-111111111111",
				JobKey:          service.JobKeySafeKeepBackup,
				Name:            "SafeKeep 全量数据定时备份",
				CronExpr:        "0 4 * * 1",
				DefaultSchedule: "0 4 * * 1",
				Enabled:         true,
				TimeoutSec:      300,
				ConfigJSON:      `{"api_base_url":"https://sk.local","has_api_key":true}`,
			},
		},
		runs: []dto.CronJobRunResp{
			{
				ID:        "run-1",
				JobID:     "11111111-1111-1111-1111-111111111111",
				Status:    "succeeded",
				OutputLog: "uploaded backup.zip",
			},
		},
		secrets: dto.CronJobSecretsResp{APIKey: "sk_secret"},
		testRes: dto.TestSafeKeepResp{
			OK: true,
			Upstreams: []dto.SafeKeepUpstreamResp{
				{UpstreamID: "u1", UpstreamName: "s3-prod", UpstreamType: "s3", Enabled: true},
			},
		},
	}

	router := gin.New()
	admin := router.Group("/api/v1/admin")
	NewCronHandler(cronSvc, auditSvc, zerolog.Nop()).RegisterAdminRoutes(admin)

	// Preview valid & invalid
	prevRec := httptest.NewRecorder()
	prevReq := httptest.NewRequest(http.MethodPost, "/api/v1/admin/cron-jobs/preview", strings.NewReader(`{"schedule":"0 4 * * 1"}`))
	prevReq.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(prevRec, prevReq)
	require.Equal(t, http.StatusOK, prevRec.Code)
	require.Contains(t, prevRec.Body.String(), `"next_runs"`)

	badPrevRec := httptest.NewRecorder()
	badPrevReq := httptest.NewRequest(http.MethodPost, "/api/v1/admin/cron-jobs/preview", strings.NewReader(`{"schedule":"bad"}`))
	badPrevReq.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(badPrevRec, badPrevReq)
	require.Equal(t, http.StatusBadRequest, badPrevRec.Code)

	// Update job records audit
	updRec := httptest.NewRecorder()
	updReq := httptest.NewRequest(http.MethodPut, "/api/v1/admin/cron-jobs/11111111-1111-1111-1111-111111111111", strings.NewReader(`{"cron_expr":"0 4 * * 1","enabled":true}`))
	updReq.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(updRec, updReq)
	require.Equal(t, http.StatusOK, updRec.Code)

	// Reveal secrets records audit
	secRec := httptest.NewRecorder()
	secReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/cron-jobs/11111111-1111-1111-1111-111111111111/secrets", nil)
	router.ServeHTTP(secRec, secReq)
	require.Equal(t, http.StatusOK, secRec.Code)
	require.Contains(t, secRec.Body.String(), `"api_key":"sk_secret"`)

	// Test SafeKeep records audit
	testRec := httptest.NewRecorder()
	testReq := httptest.NewRequest(http.MethodPost, "/api/v1/admin/cron-jobs/11111111-1111-1111-1111-111111111111/test-safekeep", strings.NewReader(`{"api_base_url":"https://sk.local"}`))
	testReq.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(testRec, testReq)
	require.Equal(t, http.StatusOK, testRec.Code)
	require.Contains(t, testRec.Body.String(), `"s3-prod"`)

	require.Len(t, auditSvc.recorded, 3)
	require.Equal(t, "cron.update", auditSvc.recorded[0].Action)
	require.Equal(t, "cron.reveal_secrets", auditSvc.recorded[1].Action)
	require.Equal(t, "cron.test_safekeep", auditSvc.recorded[2].Action)
	revealDetail, ok := auditSvc.recorded[1].Detail.(map[string]any)
	require.True(t, ok)
	require.Equal(t, service.JobKeySafeKeepBackup, revealDetail["job_key"])
	testDetail, ok := auditSvc.recorded[2].Detail.(map[string]any)
	require.True(t, ok)
	require.Equal(t, service.JobKeySafeKeepBackup, testDetail["job_key"])
}
