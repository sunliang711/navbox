package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"navbox/internal/config"
	"navbox/internal/dto"
	"navbox/internal/response"
	"navbox/internal/service"
)

type IconHandler struct {
	cfg     config.Config
	service service.IconService
}

func NewIconHandler(cfg config.Config, service service.IconService) *IconHandler {
	return &IconHandler{cfg: cfg, service: service}
}

func (h *IconHandler) RegisterAdminRoutes(rg *gin.RouterGroup) {
	rg.POST("/icons/upload", h.UploadIcon)
	rg.POST("/icons/fetch", h.FetchIcon)
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
		writeServiceError(c, err)
		return
	}
	response.OK(c, icon)
}
