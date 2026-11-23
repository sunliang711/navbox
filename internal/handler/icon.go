package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"navbox/internal/config"
	"navbox/internal/dto"
	"navbox/internal/response"
	"navbox/internal/service"
)

type IconHandler struct {
	cfg      config.Config
	service  service.IconService
	auditSvc service.AuditService
}

func NewIconHandler(cfg config.Config, service service.IconService, auditSvc service.AuditService) *IconHandler {
	return &IconHandler{cfg: cfg, service: service, auditSvc: auditSvc}
}

func (h *IconHandler) RegisterAdminRoutes(rg *gin.RouterGroup) {
	rg.GET("/icons", h.ListIcons)
	rg.POST("/icons/upload", h.UploadIcon)
	rg.POST("/icons/fetch", h.FetchIcon)
}

func (h *IconHandler) ListIcons(c *gin.Context) {
	limit := 0
	if raw := c.Query("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "参数错误")
			return
		}
		limit = parsed
	}
	offset := 0
	if raw := c.Query("offset"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "参数错误")
			return
		}
		offset = parsed
	}
	unused := false
	if raw := c.Query("unused"); raw != "" {
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "参数错误")
			return
		}
		unused = parsed
	}

	result, err := h.service.ListIcons(c.Request.Context(), dto.IconListQuery{
		Q:      c.Query("q"),
		Unused: unused,
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.OK(c, result)
}

func (h *IconHandler) UploadIcon(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, h.cfg.Upload.MaxBytes+1024*1024)

	header, err := c.FormFile("file")
	if err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "参数错误")
		return
	}
	if header.Size > h.cfg.Upload.MaxBytes {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "文件过大")
		return
	}

	file, err := header.Open()
	if err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "参数错误")
		return
	}
	defer file.Close()

	icon, err := h.service.UploadIcon(c.Request.Context(), file)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	recordAdminAudit(c, h.auditSvc, "icon.upload", "icon", icon.ID, service.AuditStatusSuccess, map[string]any{
		"file_name":  icon.FileName,
		"mime_type":  icon.MIMEType,
		"size_bytes": icon.SizeBytes,
	})
	response.OK(c, icon)
}

func (h *IconHandler) FetchIcon(c *gin.Context) {
	var req dto.FetchIconReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "参数错误")
		return
	}

	icon, err := h.service.FetchIcon(c.Request.Context(), req.URL)
	if err != nil {
		recordAdminAudit(c, h.auditSvc, "icon.fetch", "icon", "", service.AuditStatusFailure, map[string]any{
			"url": req.URL,
		})
		writeServiceError(c, err)
		return
	}
	recordAdminAudit(c, h.auditSvc, "icon.fetch", "icon", icon.ID, service.AuditStatusSuccess, map[string]any{
		"url":       req.URL,
		"file_name": icon.FileName,
	})
	response.OK(c, icon)
}
