package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"navbox/internal/config"
	"navbox/internal/dto"
	"navbox/internal/middleware"
	"navbox/internal/response"
	"navbox/internal/service"
)

type AuthHandler struct {
	cfg      config.Config
	service  service.AuthService
	auditSvc service.AuditService
}

func NewAuthHandler(cfg config.Config, service service.AuthService, auditSvc service.AuditService) *AuthHandler {
	return &AuthHandler{cfg: cfg, service: service, auditSvc: auditSvc}
}

func (h *AuthHandler) RegisterRoutes(rg *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	admin := rg.Group("/admin")
	admin.POST("/login", h.Login)
	admin.POST("/logout", authMiddleware, h.Logout)
	admin.GET("/session", authMiddleware, h.GetSession)
	admin.POST("/password", authMiddleware, h.ChangePassword)
}

func (h *AuthHandler) Login(c *gin.Context) {
	middleware.SetNoCacheHeaders(c)

	var req dto.LoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "参数错误")
		return
	}

	result, err := h.service.Login(c.Request.Context(), req.Password)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			recordAdminAudit(c, h.auditSvc, "auth.login", "session", "", service.AuditStatusFailure, map[string]any{
				"reason": "invalid_credentials",
			})
			response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "认证失败")
			return
		}
		recordAdminAudit(c, h.auditSvc, "auth.login", "session", "", service.AuditStatusFailure, map[string]any{
			"reason": "internal_error",
		})
		response.Error(c, http.StatusInternalServerError, response.CodeError, "系统繁忙，请稍后再试")
		return
	}

	middleware.SetSessionCookie(c, h.cfg, result.Token)
	recordAdminAudit(c, h.auditSvc, "auth.login", "session", "", service.AuditStatusSuccess, nil)
	response.OK(c, dto.SessionResp{Authenticated: true})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	middleware.SetNoCacheHeaders(c)

	token, _ := c.Cookie(h.cfg.Auth.CookieName)
	if err := h.service.Logout(c.Request.Context(), token); err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeError, "系统繁忙，请稍后再试")
		return
	}

	middleware.ClearSessionCookie(c, h.cfg)
	recordAdminAudit(c, h.auditSvc, "auth.logout", "session", "", service.AuditStatusSuccess, nil)
	response.OK(c, dto.SessionResp{Authenticated: false})
}

func (h *AuthHandler) GetSession(c *gin.Context) {
	middleware.SetNoCacheHeaders(c)
	response.OK(c, dto.SessionResp{Authenticated: true})
}

func (h *AuthHandler) ChangePassword(c *gin.Context) {
	middleware.SetNoCacheHeaders(c)

	var req dto.ChangePasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "参数错误")
		return
	}

	token, _ := c.Cookie(h.cfg.Auth.CookieName)
	if err := h.service.ChangePassword(c.Request.Context(), token, req.CurrentPassword, req.NewPassword); err != nil {
		switch {
		case errors.Is(err, service.ErrUnauthenticated):
			recordAdminAudit(c, h.auditSvc, "auth.change_password", "admin_setting", "1", service.AuditStatusFailure, map[string]any{"reason": "unauthenticated"})
			response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "认证失败")
		case errors.Is(err, service.ErrInvalidCredentials):
			recordAdminAudit(c, h.auditSvc, "auth.change_password", "admin_setting", "1", service.AuditStatusFailure, map[string]any{"reason": "invalid_credentials"})
			response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "认证失败")
		case errors.Is(err, service.ErrWeakPassword):
			recordAdminAudit(c, h.auditSvc, "auth.change_password", "admin_setting", "1", service.AuditStatusFailure, map[string]any{"reason": "weak_password"})
			response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "新密码强度不足")
		default:
			recordAdminAudit(c, h.auditSvc, "auth.change_password", "admin_setting", "1", service.AuditStatusFailure, map[string]any{"reason": "internal_error"})
			response.Error(c, http.StatusInternalServerError, response.CodeError, "系统繁忙，请稍后再试")
		}
		return
	}

	recordAdminAudit(c, h.auditSvc, "auth.change_password", "admin_setting", "1", service.AuditStatusSuccess, nil)
	response.OK(c, dto.SessionResp{Authenticated: true})
}
