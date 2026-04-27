package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"navbox/internal/dto"
	"navbox/internal/response"
	"navbox/internal/service"
)

type TagHandler struct {
	service service.TagService
}

func NewTagHandler(service service.TagService) *TagHandler {
	return &TagHandler{service: service}
}

func (h *TagHandler) RegisterPublicRoutes(rg *gin.RouterGroup) {
	rg.GET("/tags", h.ListTags)
	rg.GET("/config/public", h.GetPublicConfig)
}

func (h *TagHandler) RegisterAdminRoutes(rg *gin.RouterGroup) {
	rg.POST("/tags", h.CreateTag)
	rg.PUT("/tags/:id", h.UpdateTag)
	rg.DELETE("/tags/:id", h.DeleteTag)
	rg.PUT("/tags/:id/default", h.SetDefaultTag)
	rg.PUT("/tags/order", h.UpdateTagOrder)
}

func (h *TagHandler) ListTags(c *gin.Context) {
	tags, err := h.service.ListTags(c.Request.Context())
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.OK(c, tags)
}

func (h *TagHandler) GetPublicConfig(c *gin.Context) {
	cfg, err := h.service.GetPublicConfig(c.Request.Context())
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.OK(c, cfg)
}

func (h *TagHandler) CreateTag(c *gin.Context) {
	var req dto.TagSaveReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "参数错误")
		return
	}
	tag, err := h.service.CreateTag(c.Request.Context(), req)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.OK(c, tag)
}

func (h *TagHandler) UpdateTag(c *gin.Context) {
	var req dto.TagSaveReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "参数错误")
		return
	}
	tag, err := h.service.UpdateTag(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.OK(c, tag)
}

func (h *TagHandler) DeleteTag(c *gin.Context) {
	if err := h.service.DeleteTag(c.Request.Context(), c.Param("id")); err != nil {
		writeServiceError(c, err)
		return
	}
	response.OK(c, gin.H{"deleted": true})
}

func (h *TagHandler) SetDefaultTag(c *gin.Context) {
	if err := h.service.SetDefaultTag(c.Request.Context(), c.Param("id")); err != nil {
		writeServiceError(c, err)
		return
	}
	response.OK(c, gin.H{"updated": true})
}

func (h *TagHandler) UpdateTagOrder(c *gin.Context) {
	var req dto.UpdateOrderReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "参数错误")
		return
	}
	if err := h.service.UpdateTagOrder(c.Request.Context(), req); err != nil {
		writeServiceError(c, err)
		return
	}
	response.OK(c, gin.H{"updated": true})
}
