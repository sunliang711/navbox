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
	service service.ImportExportService
	logger  zerolog.Logger
}

func NewImportExportHandler(service service.ImportExportService, logger zerolog.Logger) *ImportExportHandler {
	return &ImportExportHandler{service: service, logger: logger}
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
		writeServiceError(c, err)
		return
	}

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
		writeServiceError(c, err)
		return
	}
	response.OK(c, report)
}
