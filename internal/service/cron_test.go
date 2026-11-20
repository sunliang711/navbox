package service

import (
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"navbox/internal/crypto"
	"navbox/internal/dto"
	"navbox/internal/model"
	"navbox/internal/repo"
)

type memCronRepo struct {
	mu   sync.Mutex
	jobs map[uuid.UUID]model.CronJob
	runs map[uuid.UUID][]model.CronJobRun
}

func newMemCronRepo() *memCronRepo {
	return &memCronRepo{
		jobs: make(map[uuid.UUID]model.CronJob),
		runs: make(map[uuid.UUID][]model.CronJobRun),
	}
}

func (m *memCronRepo) SeedDefaultJobs(_ context.Context, jobs []model.CronJob) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, j := range jobs {
		exists := false
		for _, cur := range m.jobs {
			if cur.JobKey == j.JobKey {
				exists = true
				break
			}
		}
		if !exists {
			if j.ID == uuid.Nil {
				j.ID = uuid.New()
			}
			m.jobs[j.ID] = j
		}
	}
	return nil
}

func (m *memCronRepo) ListJobs(_ context.Context) ([]model.CronJob, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]model.CronJob, 0, len(m.jobs))
	for _, j := range m.jobs {
		out = append(out, j)
	}
	return out, nil
}

func (m *memCronRepo) GetJobByID(_ context.Context, id uuid.UUID) (*model.CronJob, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	j, ok := m.jobs[id]
	if !ok {
		return nil, nil
	}
	cp := j
	return &cp, nil
}

func (m *memCronRepo) GetJobByKey(_ context.Context, key string) (*model.CronJob, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, j := range m.jobs {
		if j.JobKey == key {
			cp := j
			return &cp, nil
		}
	}
	return nil, nil
}

func (m *memCronRepo) UpdateJob(_ context.Context, id uuid.UUID, cronExpr string, enabled bool, timeoutSec int, configJSON string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	j, ok := m.jobs[id]
	if !ok {
		return false, nil
	}
	j.CronExpr = cronExpr
	j.Enabled = enabled
	j.TimeoutSec = timeoutSec
	j.ConfigJSON = configJSON
	j.UpdatedAt = time.Now().UTC()
	m.jobs[id] = j
	return true, nil
}

func (m *memCronRepo) UpdateJobNextRun(_ context.Context, id uuid.UUID, nextRun *time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if j, ok := m.jobs[id]; ok {
		j.NextRunAt = nextRun
		m.jobs[id] = j
	}
	return nil
}

func (m *memCronRepo) UpdateJobExecution(_ context.Context, id uuid.UUID, status string, summary string, durationMs int64, runAt time.Time, nextRun *time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if j, ok := m.jobs[id]; ok {
		j.LastStatus = status
		j.LastSummary = summary
		j.LastDurationMs = durationMs
		j.LastRunAt = &runAt
		j.NextRunAt = nextRun
		m.jobs[id] = j
	}
	return nil
}

func (m *memCronRepo) CreateRun(_ context.Context, run *model.CronJobRun) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if run.ID == uuid.Nil {
		run.ID = uuid.New()
	}
	m.runs[run.JobID] = append([]model.CronJobRun{*run}, m.runs[run.JobID]...)
	return nil
}

func (m *memCronRepo) FinishRun(_ context.Context, id uuid.UUID, status string, affectedRows int64, outputLog string, finishedAt time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for jobID, list := range m.runs {
		for i := range list {
			if list[i].ID == id {
				list[i].Status = status
				list[i].AffectedRows = affectedRows
				list[i].OutputLog = outputLog
				list[i].FinishedAt = &finishedAt
				m.runs[jobID] = list
				return nil
			}
		}
	}
	return nil
}

func (m *memCronRepo) ListRuns(_ context.Context, jobID uuid.UUID, limit int) ([]model.CronJobRun, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	list := m.runs[jobID]
	if limit > 0 && len(list) > limit {
		list = list[:limit]
	}
	return append([]model.CronJobRun(nil), list...), nil
}

func (m *memCronRepo) DeleteRunsBefore(_ context.Context, cutoff time.Time) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var deleted int64
	for jobID, list := range m.runs {
		var kept []model.CronJobRun
		for _, r := range list {
			if r.StartedAt.Before(cutoff) {
				deleted++
			} else {
				kept = append(kept, r)
			}
		}
		m.runs[jobID] = kept
	}
	return deleted, nil
}

type memAuditRepo struct {
	mu   sync.Mutex
	logs []model.AuditLog
}

func (m *memAuditRepo) Create(_ context.Context, log *model.AuditLog) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	log.ID = int64(len(m.logs) + 1)
	m.logs = append(m.logs, *log)
	return nil
}

func (m *memAuditRepo) List(_ context.Context, _ repo.AuditFilter) ([]model.AuditLog, int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]model.AuditLog(nil), m.logs...), int64(len(m.logs)), nil
}

func (m *memAuditRepo) ExportAll(ctx context.Context, f repo.AuditFilter) ([]model.AuditLog, error) {
	logs, _, err := m.List(ctx, f)
	return logs, err
}

func (m *memAuditRepo) DeleteBefore(_ context.Context, cutoff time.Time) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var kept []model.AuditLog
	var deleted int64
	for _, l := range m.logs {
		if l.CreatedAt.Before(cutoff) {
			deleted++
		} else {
			kept = append(kept, l)
		}
	}
	m.logs = kept
	return deleted, nil
}

type stubImportExportService struct {
	fileName string
	content  []byte
	err      error
	calls    int
}

func (s *stubImportExportService) Export(_ context.Context, req dto.ExportReq) (*dto.ExportArchiveResp, error) {
	s.calls++
	if len(req.SiteIDs) != 0 || len(req.TagIDs) != 0 {
		return nil, ErrInvalidInput
	}
	if s.err != nil {
		return nil, s.err
	}
	return &dto.ExportArchiveResp{
		FileName: s.fileName,
		Content:  s.content,
	}, nil
}

func (s *stubImportExportService) Import(_ context.Context, _ multipart.File) (*dto.ImportReportResp, error) {
	return &dto.ImportReportResp{}, nil
}

func newTestStaticCipher(t *testing.T) SecretCipher {
	t.Helper()
	hexKey, err := crypto.GenerateMasterKeyHex()
	require.NoError(t, err)
	raw, err := crypto.DecodeMasterKey(hexKey)
	require.NoError(t, err)
	c, err := NewStaticSecretCipher(raw)
	require.NoError(t, err)
	return c
}

func TestPreviewCron(t *testing.T) {
	times, err := PreviewCron("0 4 * * 1", 5)
	require.NoError(t, err)
	require.Len(t, times, 5)
	for i := 1; i < len(times); i++ {
		require.True(t, times[i].After(times[i-1]))
	}

	weeklyTimes, err := PreviewCron("@weekly", 3)
	require.NoError(t, err)
	require.Len(t, weeklyTimes, 3)

	_, err = PreviewCron("not-a-cron", 5)
	require.Error(t, err)

	_, err = PreviewCron("   ", 5)
	require.Error(t, err)
}

func TestNormalizeCronJobConfig(t *testing.T) {
	got, err := NormalizeCronJobConfig(JobKeyCleanupAuditLogs, json.RawMessage(`{"retention_days":30}`))
	require.NoError(t, err)
	require.Equal(t, `{"retention_days":30}`, got)

	_, err = NormalizeCronJobConfig(JobKeyCleanupAuditLogs, json.RawMessage(`{"retention_days":0}`))
	require.Error(t, err)

	got, err = NormalizeCronJobConfig(JobKeyCleanupCronRuns, json.RawMessage(`{"retention_days":60}`))
	require.NoError(t, err)
	require.Equal(t, `{"retention_days":60}`, got)
}

func TestSafeKeepBackupConfigEncryptionAndSanitization(t *testing.T) {
	cipher := newTestStaticCipher(t)

	merged, err := MergeSafeKeepBackupConfig(cipher, "", json.RawMessage(`{
		"api_base_url": "https://safekeep.example.com/",
		"api_key": "sk_test_secret_key",
		"path": "navbox-weekly",
		"targets": "prod-s3,team-dropbox",
		"mode": "sync",
		"rename": "weekly.zip",
		"skip_dedup": true,
		"skip_tls_verify": true
	}`))
	require.NoError(t, err)
	require.NotContains(t, merged, "sk_test_secret_key")

	sanitized := SanitizeSafeKeepBackupConfigJSON(merged)
	require.Contains(t, sanitized, `"has_api_key":true`)
	require.NotContains(t, sanitized, "api_key_encrypted")
	require.NotContains(t, sanitized, "sk_test_secret_key")

	revealed, err := RevealSafeKeepBackupAPIKey(cipher, merged)
	require.NoError(t, err)
	require.Equal(t, "sk_test_secret_key", revealed)

	// 传入掩码占位符时应保留既有加密密钥
	preserved, err := MergeSafeKeepBackupConfig(cipher, merged, json.RawMessage(`{
		"api_base_url": "https://safekeep.example.com",
		"api_key": "*****",
		"path": "navbox-weekly",
		"mode": "async"
	}`))
	require.NoError(t, err)
	revealedAfterPreserve, err := RevealSafeKeepBackupAPIKey(cipher, preserved)
	require.NoError(t, err)
	require.Equal(t, "sk_test_secret_key", revealedAfterPreserve)

	// 显式传空字符串清空密钥
	cleared, err := MergeSafeKeepBackupConfig(cipher, preserved, json.RawMessage(`{
		"api_base_url": "https://safekeep.example.com",
		"api_key": "",
		"mode": "sync"
	}`))
	require.NoError(t, err)
	revealedAfterClear, err := RevealSafeKeepBackupAPIKey(cipher, cleared)
	require.NoError(t, err)
	require.Empty(t, revealedAfterClear)

	// 非法参数拦截
	_, err = MergeSafeKeepBackupConfig(cipher, "", json.RawMessage(`{"api_base_url":"ftp://invalid"}`))
	require.Error(t, err)
	_, err = MergeSafeKeepBackupConfig(cipher, "", json.RawMessage(`{"mode":"unknown"}`))
	require.Error(t, err)
	_, err = MergeSafeKeepBackupConfig(cipher, "", json.RawMessage(`{"rename":"../evil.zip"}`))
	require.Error(t, err)
}

func TestCronServiceSafeKeepBackupExecutionAndTestConnection(t *testing.T) {
	var receivedAPIKey string
	var receivedMode, receivedPath, receivedTargets, receivedRename, receivedDedup, receivedFilename string
	var receivedFileBytes []byte

	mockSafeKeep := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAPIKey = r.Header.Get("X-API-Key")
		if receivedAPIKey != "sk_live_navbox_key" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":{"code":"invalid_api_key","message":"invalid API key"}}`))
			return
		}
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/upstreams":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"items":[{"upstream_id":"up-1","upstream_name":"prod-s3","upstream_type":"s3","enabled":true,"is_default":true,"priority":10}]}`))
			return
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/backups":
			require.NoError(t, r.ParseMultipartForm(10<<20))
			receivedMode = r.FormValue("mode")
			receivedPath = r.FormValue("path")
			receivedTargets = r.FormValue("targets")
			receivedRename = r.FormValue("rename")
			receivedDedup = r.FormValue("dedup")
			f, hdr, err := r.FormFile("file")
			require.NoError(t, err)
			defer f.Close()
			receivedFilename = hdr.Filename
			receivedFileBytes, err = io.ReadAll(f)
			require.NoError(t, err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"task":{"id":"task-123","status":"succeeded","original_filename":"` + hdr.Filename + `","remote_filename":"20261120-navbox.zip"}}`))
			return
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer mockSafeKeep.Close()

	cronRepo := newMemCronRepo()
	auditRepo := &memAuditRepo{}
	auditSvc := NewAuditService(auditRepo, zerolog.Nop())
	exportSvc := &stubImportExportService{
		fileName: "navbox-export-20261120040000.zip",
		content:  []byte("PK-mock-zip-content"),
	}
	cipher := newTestStaticCipher(t)

	appCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	svc := &cronService{
		logger:          zerolog.Nop(),
		cronRepo:        cronRepo,
		auditSvc:        auditSvc,
		importExportSvc: exportSvc,
		cipher:          cipher,
		safeKeepClient:  NewSafeKeepClient(),
		entryIDs:        make(map[uuid.UUID]cron.EntryID),
		running:         make(map[uuid.UUID]bool),
		appCtx:          appCtx,
		appCancel:       cancel,
	}

	ctx := context.Background()
	require.NoError(t, svc.seedDefaults(ctx))

	jobs, err := svc.ListJobs(ctx)
	require.NoError(t, err)
	require.Len(t, jobs, 3)

	var backupJob dto.CronJobResp
	for _, j := range jobs {
		if j.JobKey == JobKeySafeKeepBackup {
			backupJob = j
		}
	}
	require.Equal(t, "0 4 * * 1", backupJob.CronExpr)
	require.Equal(t, "0 4 * * 1", backupJob.DefaultSchedule)
	require.False(t, backupJob.Enabled)

	// 1. 未配置 SafeKeep 地址和密钥时不允许启用
	_, err = svc.UpdateJob(ctx, backupJob.ID, dto.UpdateCronJobReq{
		CronExpr: "0 4 * * 1",
		Enabled:  true,
	})
	require.Error(t, err)

	// 2. 配置 SafeKeep 参数并启用
	timeout := 120
	updated, err := svc.UpdateJob(ctx, backupJob.ID, dto.UpdateCronJobReq{
		CronExpr:   "0 5 * * 0",
		Enabled:    true,
		TimeoutSec: &timeout,
		Config: json.RawMessage(`{
			"api_base_url": "` + mockSafeKeep.URL + `",
			"api_key": "sk_live_navbox_key",
			"path": "navbox-backups",
			"targets": "prod-s3",
			"mode": "sync",
			"rename": "latest-navbox.zip",
			"skip_dedup": true
		}`),
	})
	require.NoError(t, err)
	require.True(t, updated.Enabled)
	require.Equal(t, "0 5 * * 0", updated.CronExpr)
	require.Equal(t, 120, updated.TimeoutSec)
	require.Contains(t, updated.ConfigJSON, `"has_api_key":true`)
	require.NotContains(t, updated.ConfigJSON, "sk_live_navbox_key")

	// 3. 测试连接获取上游列表（复用已保存密钥）
	testRes, err := svc.TestSafeKeepConnection(ctx, backupJob.ID, dto.TestSafeKeepReq{})
	require.NoError(t, err)
	require.True(t, testRes.OK)
	require.Len(t, testRes.Upstreams, 1)
	require.Equal(t, "prod-s3", testRes.Upstreams[0].UpstreamName)

	// 4. 立即执行备份任务：导出全部数据并上传 SafeKeep
	run, err := svc.RunNow(ctx, backupJob.ID, "admin")
	require.NoError(t, err)
	require.Equal(t, "succeeded", run.Status)
	require.Equal(t, int64(1), run.AffectedRows)
	require.Contains(t, run.OutputLog, "task-123")
	require.Equal(t, 1, exportSvc.calls)
	require.Equal(t, "sk_live_navbox_key", receivedAPIKey)
	require.Equal(t, "sync", receivedMode)
	require.Equal(t, "navbox-backups", receivedPath)
	require.Equal(t, "prod-s3", receivedTargets)
	require.Equal(t, "latest-navbox.zip", receivedRename)
	require.Equal(t, "false", receivedDedup)
	require.Equal(t, "navbox-export-20261120040000.zip", receivedFilename)
	require.Equal(t, []byte("PK-mock-zip-content"), receivedFileBytes)

	// 5. 审计日志已记录 cron.execute
	auditList, err := auditSvc.List(ctx, dto.AuditListQuery{})
	require.NoError(t, err)
	require.Len(t, auditList.Items, 1)
	require.Equal(t, "cron.execute", auditList.Items[0].Action)
	require.Equal(t, AuditStatusSuccess, auditList.Items[0].Status)
	require.True(t, strings.Contains(auditList.Items[0].DetailJSON, `"job_key":"safekeep_backup"`))

	// 6. 行内切换开关（不传 config）应保留加密密钥
	toggled, err := svc.UpdateJob(ctx, backupJob.ID, dto.UpdateCronJobReq{
		CronExpr:   "0 5 * * 0",
		Enabled:    false,
		TimeoutSec: &timeout,
	})
	require.NoError(t, err)
	require.False(t, toggled.Enabled)
	require.Contains(t, toggled.ConfigJSON, `"has_api_key":true`)

	secrets, err := svc.RevealSecrets(ctx, backupJob.ID)
	require.NoError(t, err)
	require.Equal(t, "sk_live_navbox_key", secrets.APIKey)
}
