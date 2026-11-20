package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"navbox/internal/dto"
	"navbox/internal/response"
	"navbox/internal/service"
)

type CronHandler struct {
	service  service.CronService
	auditSvc service.AuditService
	logger   zerolog.Logger
}

func NewCronHandler(cronSvc service.CronService, auditSvc service.AuditService, logger zerolog.Logger) *CronHandler {
	return &CronHandler{
		service:  cronSvc,
		auditSvc: auditSvc,
		logger:   logger.With().Str("component", "cron_handler").Logger(),
	}
}

func (h *CronHandler) RegisterAdminRoutes(rg *gin.RouterGroup) {
	rg.GET("/cron-jobs", h.ListJobs)
	rg.POST("/cron-jobs/preview", h.PreviewSchedule)
	rg.PUT("/cron-jobs/:id", h.UpdateJob)
	rg.POST("/cron-jobs/:id/run", h.RunJob)
	rg.GET("/cron-jobs/:id/runs", h.ListRuns)
	rg.GET("/cron-jobs/:id/secrets", h.RevealSecrets)
	rg.POST("/cron-jobs/:id/test-safekeep", h.TestSafeKeep)
}

func (h *CronHandler) ListJobs(c *gin.Context) {
	jobs, err := h.service.ListJobs(c.Request.Context())
	if err != nil {
		h.logger.Error().Err(err).Msg("List cron jobs failed")
		writeServiceError(c, err)
		return
	}
	response.OK(c, dto.CronJobListResp{Items: jobs})
}

func (h *CronHandler) PreviewSchedule(c *gin.Context) {
	var req dto.PreviewCronReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "参数错误")
		return
	}
	expr := strings.TrimSpace(req.Schedule)
	if expr == "" {
		expr = strings.TrimSpace(req.CronExpr)
	}
	times, err := service.PreviewCron(expr, 5)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.OK(c, dto.PreviewCronResp{NextRuns: times})
}

func (h *CronHandler) UpdateJob(c *gin.Context) {
	var req dto.UpdateCronJobReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "参数错误")
		return
	}

	id := c.Param("id")
	job, err := h.service.UpdateJob(c.Request.Context(), id, req)
	if err != nil {
		if !errors.Is(err, service.ErrInvalidInput) && !errors.Is(err, service.ErrNotFound) {
			h.logger.Error().Err(err).Str("job_id", id).Msg("Update cron job failed")
		}
		writeServiceError(c, err)
		return
	}

	recordAdminAudit(c, h.auditSvc, "cron.update", "cron_job", job.ID, service.AuditStatusSuccess, map[string]any{
		"job_key":     job.JobKey,
		"cron_expr":   job.CronExpr,
		"enabled":     job.Enabled,
		"timeout_sec": job.TimeoutSec,
	})
	response.OK(c, job)
}

func (h *CronHandler) RunJob(c *gin.Context) {
	id := c.Param("id")
	run, err := h.service.RunNow(c.Request.Context(), id, service.AuditActorAdmin)
	if err != nil {
		if !errors.Is(err, service.ErrInvalidInput) && !errors.Is(err, service.ErrNotFound) {
			h.logger.Error().Err(err).Str("job_id", id).Msg("Run cron job failed")
		}
		writeServiceError(c, err)
		return
	}
	response.OK(c, run)
}

func (h *CronHandler) ListRuns(c *gin.Context) {
	id := c.Param("id")
	limit, _ := strconv.Atoi(c.Query("limit"))
	runs, err := h.service.ListRuns(c.Request.Context(), id, limit)
	if err != nil {
		if !errors.Is(err, service.ErrInvalidInput) && !errors.Is(err, service.ErrNotFound) {
			h.logger.Error().Err(err).Str("job_id", id).Msg("List cron job runs failed")
		}
		writeServiceError(c, err)
		return
	}
	response.OK(c, dto.CronJobRunListResp{Items: runs})
}

func (h *CronHandler) RevealSecrets(c *gin.Context) {
	id := c.Param("id")
	secrets, err := h.service.RevealSecrets(c.Request.Context(), id)
	if err != nil {
		if !errors.Is(err, service.ErrInvalidInput) && !errors.Is(err, service.ErrNotFound) {
			h.logger.Error().Err(err).Str("job_id", id).Msg("Reveal cron job secrets failed")
		}
		writeServiceError(c, err)
		return
	}
	recordAdminAudit(c, h.auditSvc, "cron.reveal_secrets", "cron_job", id, service.AuditStatusSuccess, map[string]any{
		"job_id": id,
	})
	response.OK(c, secrets)
}

func (h *CronHandler) TestSafeKeep(c *gin.Context) {
	id := c.Param("id")
	var req dto.TestSafeKeepReq
	if c.Request.ContentLength != 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "参数错误")
			return
		}
	}

	res, err := h.service.TestSafeKeepConnection(c.Request.Context(), id, req)
	if err != nil {
		recordAdminAudit(c, h.auditSvc, "cron.test_safekeep", "cron_job", id, service.AuditStatusFailure, map[string]any{
			"api_base_url": strings.TrimSpace(req.APIBaseURL),
			"error":        err.Error(),
		})
		var vErr *service.ValidationError
		if errors.As(err, &vErr) {
			response.Error(c, http.StatusBadRequest, response.CodeBadRequest, vErr.Message)
			return
		}
		if errors.Is(err, service.ErrNotFound) || errors.Is(err, service.ErrInvalidInput) {
			writeServiceError(c, err)
			return
		}
		h.logger.Warn().Err(err).Str("job_id", id).Msg("Test SafeKeep connection failed")
		response.Error(c, http.StatusBadGateway, response.CodeError, err.Error())
		return
	}

	recordAdminAudit(c, h.auditSvc, "cron.test_safekeep", "cron_job", id, service.AuditStatusSuccess, map[string]any{
		"api_base_url":   strings.TrimSpace(req.APIBaseURL),
		"upstream_count": len(res.Upstreams),
	})
	response.OK(c, res)
}
