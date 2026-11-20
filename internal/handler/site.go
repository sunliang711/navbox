package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"navbox/internal/dto"
	"navbox/internal/response"
	"navbox/internal/service"
)

type SiteHandler struct {
	service  service.SiteService
	auditSvc service.AuditService
}

func NewSiteHandler(service service.SiteService, auditSvc service.AuditService) *SiteHandler {
	return &SiteHandler{service: service, auditSvc: auditSvc}
}

func (h *SiteHandler) RegisterPublicRoutes(rg *gin.RouterGroup) {
	rg.GET("/sites", h.ListSites)
}

func (h *SiteHandler) RegisterAdminRoutes(rg *gin.RouterGroup) {
	rg.POST("/sites", h.CreateSite)
	rg.PUT("/sites/:id", h.UpdateSite)
	rg.DELETE("/sites/:id", h.DeleteSite)
	rg.POST("/sites/batch-delete", h.BatchDeleteSites)
	rg.POST("/sites/batch-tags", h.BatchUpdateSiteTags)
	rg.PUT("/sites/order", h.UpdateSiteOrder)
}

func (h *SiteHandler) ListSites(c *gin.Context) {
	sites, err := h.service.ListSites(c.Request.Context(), dto.SiteListQuery{
		Search:   c.Query("search"),
		TagIDs:   parseQueryIDs(c, "tag_ids"),
		TagMatch: c.Query("tag_match"),
		View:     c.Query("view"),
	})
	if err != nil {
		writeServiceError(c, err)
		return
	}
	response.OK(c, sites)
}

func (h *SiteHandler) CreateSite(c *gin.Context) {
	var req dto.SiteSaveReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "参数错误")
		return
	}

	site, err := h.service.CreateSite(c.Request.Context(), req)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	recordAdminAudit(c, h.auditSvc, "site.create", "site", site.ID, service.AuditStatusSuccess, map[string]any{
		"title":       site.Title,
		"default_url": site.DefaultURL,
	})
	response.OK(c, site)
}

func (h *SiteHandler) UpdateSite(c *gin.Context) {
	var req dto.SiteSaveReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "参数错误")
		return
	}

	site, err := h.service.UpdateSite(c.Request.Context(), c.Param("id"), req)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	recordAdminAudit(c, h.auditSvc, "site.update", "site", site.ID, service.AuditStatusSuccess, map[string]any{
		"title":       site.Title,
		"default_url": site.DefaultURL,
	})
	response.OK(c, site)
}

func (h *SiteHandler) DeleteSite(c *gin.Context) {
	id := c.Param("id")
	if err := h.service.DeleteSite(c.Request.Context(), id); err != nil {
		writeServiceError(c, err)
		return
	}
	recordAdminAudit(c, h.auditSvc, "site.delete", "site", id, service.AuditStatusSuccess, nil)
	response.OK(c, gin.H{"deleted": true})
}

func (h *SiteHandler) BatchDeleteSites(c *gin.Context) {
	var req dto.BatchDeleteSitesReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "参数错误")
		return
	}
	if err := h.service.BatchDeleteSites(c.Request.Context(), req); err != nil {
		writeServiceError(c, err)
		return
	}
	recordAdminAudit(c, h.auditSvc, "site.batch_delete", "site", "", service.AuditStatusSuccess, map[string]any{
		"count":    len(req.SiteIDs),
		"site_ids": req.SiteIDs,
	})
	response.OK(c, gin.H{"deleted": true})
}

func (h *SiteHandler) BatchUpdateSiteTags(c *gin.Context) {
	var req dto.BatchSiteTagsReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "参数错误")
		return
	}
	if err := h.service.BatchUpdateSiteTags(c.Request.Context(), req); err != nil {
		writeServiceError(c, err)
		return
	}
	recordAdminAudit(c, h.auditSvc, "site.batch_tags", "site", "", service.AuditStatusSuccess, map[string]any{
		"action":     req.Action,
		"site_count": len(req.SiteIDs),
		"tag_count":  len(req.TagIDs),
	})
	response.OK(c, gin.H{"updated": true})
}

func (h *SiteHandler) UpdateSiteOrder(c *gin.Context) {
	var req dto.UpdateOrderReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "参数错误")
		return
	}
	if err := h.service.UpdateSiteOrder(c.Request.Context(), req); err != nil {
		writeServiceError(c, err)
		return
	}
	recordAdminAudit(c, h.auditSvc, "site.reorder", "site", "", service.AuditStatusSuccess, map[string]any{
		"count": len(req.Items),
	})
	response.OK(c, gin.H{"updated": true})
}

func parseQueryIDs(c *gin.Context, name string) []string {
	values := c.QueryArray(name)
	if len(values) == 0 {
		raw := c.Query(name)
		if raw == "" {
			return nil
		}
		values = strings.Split(raw, ",")
	}

	result := make([]string, 0, len(values))
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			part = strings.TrimSpace(part)
			if part != "" {
				result = append(result, part)
			}
		}
	}
	return result
}

func writeServiceError(c *gin.Context, err error) {
	var vErr *service.ValidationError
	if errors.As(err, &vErr) {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, vErr.Message)
		return
	}
	switch {
	case errors.Is(err, service.ErrInvalidInput):
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "参数错误")
	case errors.Is(err, service.ErrNotFound):
		response.Error(c, http.StatusNotFound, response.CodeNotFound, "not found")
	default:
		response.Error(c, http.StatusInternalServerError, response.CodeError, "系统繁忙，请稍后再试")
	}
}
