package handler

import (
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"navbox/internal/dto"
	"navbox/internal/response"
	"navbox/internal/service"
)

type ImportExportHandler struct {
	service  service.ImportExportService
	auditSvc service.AuditService
	logger   zerolog.Logger
}

func NewImportExportHandler(service service.ImportExportService, auditSvc service.AuditService, logger zerolog.Logger) *ImportExportHandler {
	return &ImportExportHandler{service: service, auditSvc: auditSvc, logger: logger}
}

func (h *ImportExportHandler) RegisterAdminRoutes(rg *gin.RouterGroup) {
	rg.POST("/export", h.Export)
	rg.POST("/import", h.Import)
}

func (h *ImportExportHandler) Export(c *gin.Context) {
	var req dto.ExportReq
	if c.Request.ContentLength != 0 {
		if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
			response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "参数错误")
			return
		}
	}

	archive, err := h.service.Export(c.Request.Context(), req)
	if err != nil {
		h.logger.Error().Err(err).Msg("Export config failed")
		recordAdminAudit(c, h.auditSvc, "data.export", "archive", "", service.AuditStatusFailure, map[string]any{
			"site_filter_count": len(req.SiteIDs),
			"tag_filter_count":  len(req.TagIDs),
		})
		writeServiceError(c, err)
		return
	}

	recordAdminAudit(c, h.auditSvc, "data.export", "archive", archive.FileName, service.AuditStatusSuccess, map[string]any{
		"file_name":         archive.FileName,
		"size_bytes":        len(archive.Content),
		"site_filter_count": len(req.SiteIDs),
		"tag_filter_count":  len(req.TagIDs),
	})
	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", `attachment; filename="`+archive.FileName+`"`)
	c.Data(http.StatusOK, "application/zip", archive.Content)
}

func (h *ImportExportHandler) Import(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, service.MaxImportArchiveBytes+1024*1024)

	header, err := c.FormFile("file")
	if err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "参数错误")
		return
	}
	if header.Size > service.MaxImportArchiveBytes {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "文件过大")
		return
	}

	file, err := header.Open()
	if err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "参数错误")
		return
	}
	defer file.Close()

	report, err := h.service.Import(c.Request.Context(), file)
	if err != nil {
		h.logger.Error().Err(err).Msg("Import config failed")
		recordAdminAudit(c, h.auditSvc, "data.import", "archive", header.Filename, service.AuditStatusFailure, map[string]any{
			"file_name":  header.Filename,
			"size_bytes": header.Size,
		})
		writeServiceError(c, err)
		return
	}
	recordAdminAudit(c, h.auditSvc, "data.import", "archive", header.Filename, service.AuditStatusSuccess, map[string]any{
		"file_name": header.Filename,
		"imported":  report.Imported,
		"skipped":   report.Skipped,
	})
	response.OK(c, report)
}
