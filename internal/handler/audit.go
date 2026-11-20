package handler

import (
	"encoding/csv"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"navbox/internal/dto"
	"navbox/internal/response"
	"navbox/internal/service"
)

type AuditHandler struct {
	service service.AuditService
	logger  zerolog.Logger
}

func NewAuditHandler(service service.AuditService, logger zerolog.Logger) *AuditHandler {
	return &AuditHandler{
		service: service,
		logger:  logger.With().Str("component", "audit_handler").Logger(),
	}
}

func (h *AuditHandler) RegisterAdminRoutes(rg *gin.RouterGroup) {
	rg.GET("/audit-logs", h.List)
	rg.GET("/audit-logs/export", h.Export)
}

func (h *AuditHandler) List(c *gin.Context) {
	query, err := parseAuditListQuery(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, err.Error())
		return
	}
	resp, err := h.service.List(c.Request.Context(), query)
	if err != nil {
		h.logger.Error().Err(err).Msg("List audit logs failed")
		writeServiceError(c, err)
		return
	}
	response.OK(c, resp)
}

func (h *AuditHandler) Export(c *gin.Context) {
	query, err := parseAuditListQuery(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, err.Error())
		return
	}
	logs, err := h.service.ExportAll(c.Request.Context(), query)
	if err != nil {
		h.logger.Error().Err(err).Msg("Export audit logs failed")
		writeServiceError(c, err)
		return
	}

	format := strings.ToLower(strings.TrimSpace(c.Query("format")))
	if format == "json" {
		c.Header("Content-Type", "application/json; charset=utf-8")
		c.Header("Content-Disposition", `attachment; filename="audit-logs.json"`)
		c.Status(http.StatusOK)
		_ = json.NewEncoder(c.Writer).Encode(logs)
		return
	}

	c.Header("Content-Type", "text/csv; charset=utf-8")
	c.Header("Content-Disposition", `attachment; filename="audit-logs.csv"`)
	c.Status(http.StatusOK)
	cw := csv.NewWriter(c.Writer)
	_ = cw.Write([]string{"id", "created_at", "actor_type", "actor_name", "action", "resource_type", "resource_id", "status", "ip_address", "request_id", "detail_json"})
	for _, l := range logs {
		_ = cw.Write([]string{
			strconv.FormatInt(l.ID, 10),
			l.CreatedAt.Format(time.RFC3339),
			l.ActorType,
			l.ActorName,
			l.Action,
			l.ResourceType,
			l.ResourceID,
			l.Status,
			l.IPAddress,
			l.RequestID,
			l.DetailJSON,
		})
	}
	cw.Flush()
}

func parseAuditListQuery(c *gin.Context) (dto.AuditListQuery, error) {
	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))
	q := dto.AuditListQuery{
		ActorType: strings.TrimSpace(c.Query("actor_type")),
		Action:    strings.TrimSpace(c.Query("action")),
		Status:    strings.TrimSpace(c.Query("status")),
		Query:     strings.TrimSpace(c.Query("q")),
		Limit:     limit,
		Offset:    offset,
	}
	if fromStr := strings.TrimSpace(c.Query("from")); fromStr != "" {
		t, err := time.Parse(time.RFC3339, fromStr)
		if err != nil {
			return dto.AuditListQuery{}, &service.ValidationError{Message: "开始时间格式无效"}
		}
		q.From = &t
	}
	if toStr := strings.TrimSpace(c.Query("to")); toStr != "" {
		t, err := time.Parse(time.RFC3339, toStr)
		if err != nil {
			return dto.AuditListQuery{}, &service.ValidationError{Message: "结束时间格式无效"}
		}
		q.To = &t
	}
	if q.From != nil && q.To != nil && q.From.After(*q.To) {
		return dto.AuditListQuery{}, &service.ValidationError{Message: "开始时间不能晚于结束时间"}
	}
	return q, nil
}
