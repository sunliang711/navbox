package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog"
	"go.uber.org/fx"

	"navbox/internal/config"
	"navbox/internal/dto"
	"navbox/internal/model"
	"navbox/internal/repo"
)

const (
	JobKeySafeKeepBackup   = "safekeep_backup"
	JobKeyCleanupAuditLogs = "cleanup_audit_logs"
	JobKeyCleanupCronRuns  = "cleanup_cron_runs"
)

var cronParser = cron.NewParser(
	cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor,
)

type cronJobMeta struct {
	DefaultSchedule string
	DefaultTimeout  int
}

var builtinCronJobMeta = map[string]cronJobMeta{
	JobKeySafeKeepBackup:   {DefaultSchedule: "0 4 * * 1", DefaultTimeout: 300},
	JobKeyCleanupAuditLogs: {DefaultSchedule: "0 2 * * *", DefaultTimeout: 300},
	JobKeyCleanupCronRuns:  {DefaultSchedule: "0 3 * * *", DefaultTimeout: 300},
}

// DefaultCronScheduleFor 返回内置定时任务的默认 Cron 表达式。
func DefaultCronScheduleFor(jobKey string) string {
	if m, ok := builtinCronJobMeta[jobKey]; ok {
		return m.DefaultSchedule
	}
	return "0 4 * * 1"
}

// DefaultCronTimeoutFor 返回内置定时任务的默认超时秒数。
func DefaultCronTimeoutFor(jobKey string) int {
	if m, ok := builtinCronJobMeta[jobKey]; ok && m.DefaultTimeout > 0 {
		return m.DefaultTimeout
	}
	return 300
}

// PreviewCron 校验 Cron 表达式并返回未来 count 次本地触发时间。
func PreviewCron(expr string, count int) ([]time.Time, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return nil, newValidationError("cron 表达式不能为空")
	}
	sched, err := cronParser.Parse(expr)
	if err != nil {
		return nil, newValidationError(fmt.Sprintf("cron 表达式无效：%v", err))
	}
	if count <= 0 {
		count = 5
	}
	out := make([]time.Time, 0, count)
	at := time.Now()
	for i := 0; i < count; i++ {
		at = sched.Next(at)
		if at.IsZero() {
			break
		}
		out = append(out, at)
	}
	return out, nil
}

// NormalizeCronJobConfig 校验并规范化非 SafeKeep 内置任务的配置 JSON。
func NormalizeCronJobConfig(jobKey string, incoming json.RawMessage) (string, error) {
	var raw map[string]any
	if len(incoming) > 0 {
		if err := json.Unmarshal(incoming, &raw); err != nil {
			return "", newValidationError(fmt.Sprintf("任务配置 JSON 格式无效：%v", err))
		}
	}
	switch jobKey {
	case JobKeyCleanupAuditLogs:
		days := intFromMap(raw, "retention_days", 90)
		if days < 1 || days > 36500 {
			return "", newValidationError("retention_days 必须在 1 到 36500 之间")
		}
		b, _ := json.Marshal(map[string]int{"retention_days": days})
		return string(b), nil

	case JobKeyCleanupCronRuns:
		days := intFromMap(raw, "retention_days", 180)
		if days < 1 || days > 36500 {
			return "", newValidationError("retention_days 必须在 1 到 36500 之间")
		}
		b, _ := json.Marshal(map[string]int{"retention_days": days})
		return string(b), nil

	default:
		if len(incoming) == 0 {
			return "{}", nil
		}
		return string(incoming), nil
	}
}

type CronService interface {
	ListJobs(ctx context.Context) ([]dto.CronJobResp, error)
	UpdateJob(ctx context.Context, id string, req dto.UpdateCronJobReq) (dto.CronJobResp, error)
	RunNow(ctx context.Context, id string, actorName string) (dto.CronJobRunResp, error)
	ListRuns(ctx context.Context, id string, limit int) ([]dto.CronJobRunResp, error)
	RevealSecrets(ctx context.Context, id string) (dto.CronJobSecretsResp, error)
	TestSafeKeepConnection(ctx context.Context, id string, req dto.TestSafeKeepReq) (dto.TestSafeKeepResp, error)
	ReloadAll(ctx context.Context) error
}

type CronServiceParams struct {
	fx.In

	Lifecycle       fx.Lifecycle
	Config          config.Config
	Logger          zerolog.Logger
	CronRepo        repo.CronRepo
	AuthRepo        repo.AuthRepo
	AuditService    AuditService
	ImportExportSvc ImportExportService
	SecretCipher    SecretCipher
}

type cronService struct {
	cfg             config.Config
	logger          zerolog.Logger
	cronRepo        repo.CronRepo
	authRepo        repo.AuthRepo
	auditSvc        AuditService
	importExportSvc ImportExportService
	cipher          SecretCipher
	safeKeepClient  *SafeKeepClient

	mu        sync.Mutex
	c         *cron.Cron
	entryIDs  map[uuid.UUID]cron.EntryID
	running   map[uuid.UUID]bool
	appCtx    context.Context
	appCancel context.CancelFunc
}

func NewCronService(params CronServiceParams) CronService {
	appCtx, appCancel := context.WithCancel(context.Background())
	svc := &cronService{
		cfg:             params.Config,
		logger:          params.Logger.With().Str("component", "cron").Logger(),
		cronRepo:        params.CronRepo,
		authRepo:        params.AuthRepo,
		auditSvc:        params.AuditService,
		importExportSvc: params.ImportExportSvc,
		cipher:          params.SecretCipher,
		safeKeepClient:  NewSafeKeepClient(),
		entryIDs:        make(map[uuid.UUID]cron.EntryID),
		running:         make(map[uuid.UUID]bool),
		appCtx:          appCtx,
		appCancel:       appCancel,
	}

	params.Lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if params.Config.Restore.Enabled() {
				svc.logger.Info().Msg("Restore mode enabled, skipping cron scheduler startup")
				return nil
			}
			if err := svc.seedDefaults(ctx); err != nil {
				return err
			}
			return svc.start(ctx)
		},
		OnStop: func(ctx context.Context) error {
			svc.stop()
			return nil
		},
	})

	return svc
}

func defaultCronJobs() []model.CronJob {
	now := time.Now().UTC()
	return []model.CronJob{
		{
			Base:        model.Base{ID: uuid.New(), CreatedAt: now, UpdatedAt: now},
			JobKey:      JobKeySafeKeepBackup,
			Name:        "SafeKeep 全量数据定时备份",
			Description: "定期导出全部导航数据与图标（zip 压缩包），并通过 SafeKeep API 自动上传备份。默认每周执行一次，支持自定义周期与目标上游。",
			CronExpr:    DefaultCronScheduleFor(JobKeySafeKeepBackup),
			Enabled:     false,
			TimeoutSec:  DefaultCronTimeoutFor(JobKeySafeKeepBackup),
			ConfigJSON:  `{"api_base_url":"","targets":"","path":"navbox-backups","mode":"sync","rename":"","skip_dedup":false,"skip_tls_verify":false}`,
			LastStatus:  "never",
		},
		{
			Base:        model.Base{ID: uuid.New(), CreatedAt: now, UpdatedAt: now},
			JobKey:      JobKeyCleanupAuditLogs,
			Name:        "清理过期审计日志",
			Description: "定期删除超过保留天数的历史审计日志，控制数据库体积。",
			CronExpr:    DefaultCronScheduleFor(JobKeyCleanupAuditLogs),
			Enabled:     true,
			TimeoutSec:  DefaultCronTimeoutFor(JobKeyCleanupAuditLogs),
			ConfigJSON:  `{"retention_days":90}`,
			LastStatus:  "never",
		},
		{
			Base:        model.Base{ID: uuid.New(), CreatedAt: now, UpdatedAt: now},
			JobKey:      JobKeyCleanupCronRuns,
			Name:        "清理过期任务记录与会话",
			Description: "定期清理超过保留天数的定时任务执行历史以及已过期的管理员会话。",
			CronExpr:    DefaultCronScheduleFor(JobKeyCleanupCronRuns),
			Enabled:     true,
			TimeoutSec:  DefaultCronTimeoutFor(JobKeyCleanupCronRuns),
			ConfigJSON:  `{"retention_days":180}`,
			LastStatus:  "never",
		},
	}
}

func (s *cronService) seedDefaults(ctx context.Context) error {
	return s.cronRepo.SeedDefaultJobs(ctx, defaultCronJobs())
}

func (s *cronService) start(ctx context.Context) error {
	s.mu.Lock()
	s.c = cron.New(cron.WithParser(cronParser))
	s.c.Start()
	s.mu.Unlock()

	if err := s.ReloadAll(ctx); err != nil {
		return err
	}
	s.logger.Info().Msg("Cron scheduler started")
	return nil
}

func (s *cronService) stop() {
	s.appCancel()
	s.mu.Lock()
	c := s.c
	s.c = nil
	s.mu.Unlock()
	if c != nil {
		ctx := c.Stop()
		<-ctx.Done()
		s.logger.Info().Msg("Cron scheduler stopped")
	}
}

func (s *cronService) ReloadAll(ctx context.Context) error {
	jobs, err := s.cronRepo.ListJobs(ctx)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.c == nil {
		return nil
	}
	for id, entryID := range s.entryIDs {
		s.c.Remove(entryID)
		delete(s.entryIDs, id)
	}

	for _, job := range jobs {
		if !job.Enabled {
			_ = s.cronRepo.UpdateJobNextRun(ctx, job.ID, nil)
			continue
		}
		j := job
		entryID, err := s.c.AddFunc(j.CronExpr, func() {
			s.executeJob(s.appCtx, j.ID, "schedule", "scheduler")
		})
		if err != nil {
			s.logger.Error().Err(err).Str("job_key", j.JobKey).Str("cron_expr", j.CronExpr).Msg("Register cron job failed")
			continue
		}
		s.entryIDs[j.ID] = entryID
		entry := s.c.Entry(entryID)
		if !entry.Next.IsZero() {
			nxt := entry.Next.UTC()
			_ = s.cronRepo.UpdateJobNextRun(ctx, j.ID, &nxt)
		}
	}
	return nil
}

func (s *cronService) ListJobs(ctx context.Context) ([]dto.CronJobResp, error) {
	jobs, err := s.cronRepo.ListJobs(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]dto.CronJobResp, 0, len(jobs))
	for _, job := range jobs {
		items = append(items, mapCronJobResp(job))
	}
	return items, nil
}

func (s *cronService) UpdateJob(ctx context.Context, rawID string, req dto.UpdateCronJobReq) (dto.CronJobResp, error) {
	id, err := parseUUID(rawID)
	if err != nil {
		return dto.CronJobResp{}, err
	}
	existing, err := s.cronRepo.GetJobByID(ctx, id)
	if err != nil {
		return dto.CronJobResp{}, err
	}
	if existing == nil {
		return dto.CronJobResp{}, ErrNotFound
	}

	cronExpr := strings.TrimSpace(req.CronExpr)
	if _, err := PreviewCron(cronExpr, 1); err != nil {
		return dto.CronJobResp{}, err
	}

	timeoutSec := existing.TimeoutSec
	if timeoutSec <= 0 {
		timeoutSec = DefaultCronTimeoutFor(existing.JobKey)
	}
	if req.TimeoutSec != nil {
		if *req.TimeoutSec < 1 || *req.TimeoutSec > 86400 {
			return dto.CronJobResp{}, newValidationError("timeout_sec 必须在 1 到 86400 秒之间")
		}
		timeoutSec = *req.TimeoutSec
	}

	rawCfg := strings.TrimSpace(string(req.Config))
	cfgStr := existing.ConfigJSON
	if rawCfg != "" && rawCfg != "null" {
		if existing.JobKey == JobKeySafeKeepBackup {
			cfgStr, err = MergeSafeKeepBackupConfig(s.cipher, existing.ConfigJSON, req.Config)
			if err != nil {
				return dto.CronJobResp{}, err
			}
		} else {
			cfgStr, err = NormalizeCronJobConfig(existing.JobKey, req.Config)
			if err != nil {
				return dto.CronJobResp{}, err
			}
		}
	}

	if req.Enabled && existing.JobKey == JobKeySafeKeepBackup {
		parsed := ParseSafeKeepBackupConfig(cfgStr)
		if strings.TrimSpace(parsed.APIBaseURL) == "" || parsed.APIKeyEncrypted == "" {
			return dto.CronJobResp{}, newValidationError("启用 SafeKeep 备份任务前请先配置 SafeKeep API 地址和 API Key")
		}
	}

	updated, err := s.cronRepo.UpdateJob(ctx, id, cronExpr, req.Enabled, timeoutSec, cfgStr)
	if err != nil {
		return dto.CronJobResp{}, err
	}
	if !updated {
		return dto.CronJobResp{}, ErrNotFound
	}

	if err := s.ReloadAll(ctx); err != nil {
		s.logger.Error().Err(err).Msg("Reload cron scheduler failed after update")
	}

	fresh, err := s.cronRepo.GetJobByID(ctx, id)
	if err != nil {
		return dto.CronJobResp{}, err
	}
	if fresh == nil {
		return dto.CronJobResp{}, ErrNotFound
	}
	return mapCronJobResp(*fresh), nil
}

func (s *cronService) RunNow(ctx context.Context, rawID string, actorName string) (dto.CronJobRunResp, error) {
	id, err := parseUUID(rawID)
	if err != nil {
		return dto.CronJobRunResp{}, err
	}
	job, err := s.cronRepo.GetJobByID(ctx, id)
	if err != nil {
		return dto.CronJobRunResp{}, err
	}
	if job == nil {
		return dto.CronJobRunResp{}, ErrNotFound
	}
	if strings.TrimSpace(actorName) == "" {
		actorName = AuditActorAdmin
	}

	run := s.executeJob(s.appCtx, job.ID, "manual", actorName)
	return mapCronJobRunResp(*run), nil
}

func (s *cronService) ListRuns(ctx context.Context, rawID string, limit int) ([]dto.CronJobRunResp, error) {
	id, err := parseUUID(rawID)
	if err != nil {
		return nil, err
	}
	job, err := s.cronRepo.GetJobByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if job == nil {
		return nil, ErrNotFound
	}
	runs, err := s.cronRepo.ListRuns(ctx, id, limit)
	if err != nil {
		return nil, err
	}
	items := make([]dto.CronJobRunResp, 0, len(runs))
	for _, run := range runs {
		items = append(items, mapCronJobRunResp(run))
	}
	return items, nil
}

func (s *cronService) RevealSecrets(ctx context.Context, rawID string) (dto.CronJobSecretsResp, error) {
	id, err := parseUUID(rawID)
	if err != nil {
		return dto.CronJobSecretsResp{}, err
	}
	job, err := s.cronRepo.GetJobByID(ctx, id)
	if err != nil {
		return dto.CronJobSecretsResp{}, err
	}
	if job == nil {
		return dto.CronJobSecretsResp{}, ErrNotFound
	}
	if job.JobKey != JobKeySafeKeepBackup {
		return dto.CronJobSecretsResp{}, nil
	}
	apiKey, err := RevealSafeKeepBackupAPIKey(s.cipher, job.ConfigJSON)
	if err != nil {
		return dto.CronJobSecretsResp{}, err
	}
	return dto.CronJobSecretsResp{APIKey: apiKey}, nil
}

func (s *cronService) TestSafeKeepConnection(ctx context.Context, rawID string, req dto.TestSafeKeepReq) (dto.TestSafeKeepResp, error) {
	id, err := parseUUID(rawID)
	if err != nil {
		return dto.TestSafeKeepResp{}, err
	}
	job, err := s.cronRepo.GetJobByID(ctx, id)
	if err != nil {
		return dto.TestSafeKeepResp{}, err
	}
	if job == nil {
		return dto.TestSafeKeepResp{}, ErrNotFound
	}
	if job.JobKey != JobKeySafeKeepBackup {
		return dto.TestSafeKeepResp{}, newValidationError("当前任务不支持 SafeKeep 连接测试")
	}

	existing := ParseSafeKeepBackupConfig(job.ConfigJSON)
	baseURL := strings.TrimSpace(req.APIBaseURL)
	if baseURL == "" {
		baseURL = existing.APIBaseURL
	}

	apiKey := ""
	if req.APIKey != nil {
		val := strings.TrimSpace(*req.APIKey)
		if val != "" && !isMaskedSecretPlaceholder(val) {
			apiKey = val
		}
	}
	if apiKey == "" && existing.APIKeyEncrypted != "" {
		decrypted, decErr := RevealSafeKeepBackupAPIKey(s.cipher, job.ConfigJSON)
		if decErr != nil {
			return dto.TestSafeKeepResp{}, decErr
		}
		apiKey = decrypted
	}

	skipTLS := existing.SkipTLSVerify
	if req.SkipTLSVerify != nil {
		skipTLS = *req.SkipTLSVerify
	}

	testCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	upstreams, err := s.safeKeepClient.ListUpstreams(testCtx, baseURL, apiKey, skipTLS)
	if err != nil {
		return dto.TestSafeKeepResp{}, err
	}
	return dto.TestSafeKeepResp{
		OK:        true,
		Upstreams: upstreams,
	}, nil
}

func (s *cronService) executeJob(ctx context.Context, jobID uuid.UUID, triggerType, triggeredBy string) *model.CronJobRun {
	s.mu.Lock()
	if s.running[jobID] {
		s.mu.Unlock()
		now := time.Now().UTC()
		run := &model.CronJobRun{
			ID:          uuid.New(),
			JobID:       jobID,
			TriggerType: triggerType,
			TriggeredBy: triggeredBy,
			Status:      "skipped",
			OutputLog:   "上一次执行尚未结束，跳过本次",
			StartedAt:   now,
			FinishedAt:  &now,
		}
		_ = s.cronRepo.CreateRun(ctx, run)
		return run
	}
	s.running[jobID] = true
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		delete(s.running, jobID)
		s.mu.Unlock()
	}()

	job, err := s.cronRepo.GetJobByID(ctx, jobID)
	if err != nil || job == nil {
		msg := "cron job not found"
		if err != nil {
			msg = err.Error()
		}
		return &model.CronJobRun{
			ID:          uuid.New(),
			JobID:       jobID,
			TriggerType: triggerType,
			TriggeredBy: triggeredBy,
			Status:      "failed",
			OutputLog:   msg,
			StartedAt:   time.Now().UTC(),
		}
	}

	timeoutSec := job.TimeoutSec
	if timeoutSec <= 0 {
		timeoutSec = DefaultCronTimeoutFor(job.JobKey)
	}
	runCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
	defer cancel()

	start := time.Now().UTC()
	run := &model.CronJobRun{
		ID:          uuid.New(),
		JobID:       job.ID,
		TriggerType: triggerType,
		TriggeredBy: triggeredBy,
		Status:      "running",
		StartedAt:   start,
	}
	if err := s.cronRepo.CreateRun(runCtx, run); err != nil {
		s.logger.Error().Err(err).Str("job_key", job.JobKey).Msg("Create cron job run failed")
	}

	affected, summary, execErr := s.dispatchJob(runCtx, job)
	fin := time.Now().UTC()
	durationMs := fin.Sub(start).Milliseconds()

	status := "succeeded"
	if execErr != nil {
		status = "failed"
		if errors.Is(execErr, context.DeadlineExceeded) || runCtx.Err() == context.DeadlineExceeded {
			status = "timeout"
		}
		if summary == "" {
			summary = execErr.Error()
		} else {
			summary = summary + "; err: " + execErr.Error()
		}
		s.logger.Error().Err(execErr).Str("job_key", job.JobKey).Str("status", status).Msg("Cron job execution failed")
	} else {
		s.logger.Info().Str("job_key", job.JobKey).Int64("affected_rows", affected).Int64("duration_ms", durationMs).Msg("Cron job execution succeeded")
	}

	run.Status = status
	run.AffectedRows = affected
	run.OutputLog = summary
	run.FinishedAt = &fin

	finishCtx, cancelFinish := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancelFinish()
	_ = s.cronRepo.FinishRun(finishCtx, run.ID, status, affected, summary, fin)

	var nextRun *time.Time
	s.mu.Lock()
	if s.c != nil {
		if entryID, ok := s.entryIDs[job.ID]; ok {
			nxt := s.c.Entry(entryID).Next
			if !nxt.IsZero() {
				u := nxt.UTC()
				nextRun = &u
			}
		}
	}
	s.mu.Unlock()

	_ = s.cronRepo.UpdateJobExecution(finishCtx, job.ID, status, summary, durationMs, start, nextRun)

	if s.auditSvc != nil {
		actorType := AuditActorSystem
		if triggerType == "manual" {
			actorType = AuditActorAdmin
		}
		auditStatus := AuditStatusSuccess
		if execErr != nil {
			auditStatus = AuditStatusFailure
		}
		s.auditSvc.Record(finishCtx, AuditEvent{
			ActorType:    actorType,
			ActorID:      triggeredBy,
			ActorName:    triggeredBy,
			Action:       "cron.execute",
			ResourceType: "cron_job",
			ResourceID:   job.ID.String(),
			Status:       auditStatus,
			Detail: map[string]any{
				"job_key":       job.JobKey,
				"trigger_type":  triggerType,
				"status":        status,
				"affected_rows": affected,
				"summary":       summary,
				"duration_ms":   durationMs,
			},
		})
	}

	return run
}

func (s *cronService) dispatchJob(ctx context.Context, job *model.CronJob) (int64, string, error) {
	var cfg map[string]any
	_ = json.Unmarshal([]byte(job.ConfigJSON), &cfg)

	switch job.JobKey {
	case JobKeySafeKeepBackup:
		return s.runSafeKeepBackup(ctx, job.ConfigJSON)

	case JobKeyCleanupAuditLogs:
		days := intFromMap(cfg, "retention_days", 90)
		if days < 1 {
			days = 1
		}
		cutoff := time.Now().UTC().AddDate(0, 0, -days)
		n, err := s.auditSvc.CleanupBefore(ctx, cutoff)
		if err != nil {
			return 0, "", err
		}
		return n, fmt.Sprintf("deleted %d audit logs older than %d days", n, days), nil

	case JobKeyCleanupCronRuns:
		days := intFromMap(cfg, "retention_days", 180)
		if days < 1 {
			days = 1
		}
		cutoff := time.Now().UTC().AddDate(0, 0, -days)
		n, err := s.cronRepo.DeleteRunsBefore(ctx, cutoff)
		if err != nil {
			return 0, "", err
		}
		if s.authRepo != nil {
			if err := s.authRepo.DeleteExpiredSessions(ctx, time.Now().UTC()); err != nil {
				return n, fmt.Sprintf("deleted %d cron runs older than %d days", n, days), err
			}
		}
		return n, fmt.Sprintf("deleted %d cron runs older than %d days and expired sessions", n, days), nil

	default:
		return 0, "", fmt.Errorf("unknown job_key %q", job.JobKey)
	}
}

func (s *cronService) runSafeKeepBackup(ctx context.Context, rawCfg string) (int64, string, error) {
	if s.importExportSvc == nil {
		return 0, "", errors.New("import/export service is not configured")
	}
	if s.cipher == nil {
		return 0, "", errors.New("secret cipher is not configured")
	}

	cfg := ParseSafeKeepBackupConfig(rawCfg)
	if strings.TrimSpace(cfg.APIBaseURL) == "" {
		return 0, "", errors.New("SafeKeep API 地址未配置")
	}
	apiKey, err := RevealSafeKeepBackupAPIKey(s.cipher, rawCfg)
	if err != nil {
		return 0, "", fmt.Errorf("decrypt safekeep api_key: %w", err)
	}
	if strings.TrimSpace(apiKey) == "" {
		return 0, "", errors.New("SafeKeep API Key 未配置")
	}

	// 导出全部数据（网站、Tag、关联关系与本地图标）
	archive, err := s.importExportSvc.Export(ctx, dto.ExportReq{})
	if err != nil {
		return 0, "", fmt.Errorf("export all navbox data: %w", err)
	}

	res, err := s.safeKeepClient.UploadBackup(ctx, SafeKeepUploadRequest{
		APIBaseURL:    cfg.APIBaseURL,
		APIKey:        apiKey,
		FileName:      archive.FileName,
		Content:       archive.Content,
		Mode:          cfg.Mode,
		Path:          cfg.Path,
		Targets:       cfg.Targets,
		Rename:        cfg.Rename,
		SkipDedup:     cfg.SkipDedup,
		SkipTLSVerify: cfg.SkipTLSVerify,
		Metadata: map[string]any{
			"source":   "navbox_cron_backup",
			"app":      "navbox",
			"filename": archive.FileName,
			"size":     len(archive.Content),
		},
	})
	if err != nil {
		return 0, "", err
	}

	dedupNote := ""
	if res.Deduplicated {
		dedupNote = ", deduplicated"
	}
	return 1, fmt.Sprintf(
		"uploaded %s (%d bytes) to %s -> task %s (%s%s)",
		res.StoredFilename, len(archive.Content), res.Endpoint, res.TaskID, res.Status, dedupNote,
	), nil
}

func mapCronJobResp(job model.CronJob) dto.CronJobResp {
	timeoutSec := job.TimeoutSec
	if timeoutSec <= 0 {
		timeoutSec = DefaultCronTimeoutFor(job.JobKey)
	}
	nextRuns, err := PreviewCron(job.CronExpr, 5)
	if err != nil {
		nextRuns = []time.Time{}
	}
	cfgJSON := job.ConfigJSON
	if job.JobKey == JobKeySafeKeepBackup {
		cfgJSON = SanitizeSafeKeepBackupConfigJSON(cfgJSON)
	}
	return dto.CronJobResp{
		ID:              job.ID.String(),
		JobKey:          job.JobKey,
		Name:            job.Name,
		Description:     job.Description,
		CronExpr:        job.CronExpr,
		DefaultSchedule: DefaultCronScheduleFor(job.JobKey),
		Enabled:         job.Enabled,
		TimeoutSec:      timeoutSec,
		ConfigJSON:      cfgJSON,
		LastRunAt:       job.LastRunAt,
		LastStatus:      job.LastStatus,
		LastDurationMs:  job.LastDurationMs,
		LastSummary:     job.LastSummary,
		NextRunAt:       job.NextRunAt,
		NextRuns:        nextRuns,
		UpdatedAt:       job.UpdatedAt,
	}
}

func mapCronJobRunResp(run model.CronJobRun) dto.CronJobRunResp {
	return dto.CronJobRunResp{
		ID:           run.ID.String(),
		JobID:        run.JobID.String(),
		TriggerType:  run.TriggerType,
		TriggeredBy:  run.TriggeredBy,
		Status:       run.Status,
		AffectedRows: run.AffectedRows,
		OutputLog:    run.OutputLog,
		StartedAt:    run.StartedAt,
		FinishedAt:   run.FinishedAt,
	}
}

func intFromMap(m map[string]any, key string, fallback int) int {
	if m == nil {
		return fallback
	}
	v, ok := m[key]
	if !ok {
		return fallback
	}
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	default:
		return fallback
	}
}
