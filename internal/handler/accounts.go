package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"navbox/internal/dto"
	"navbox/internal/middleware"
	"navbox/internal/response"
	"navbox/internal/service"
)

func writeAccountError(c *gin.Context, err error) {
	var validation *service.ValidationError
	switch {
	case errors.Is(err, service.ErrUnauthenticated), errors.Is(err, service.ErrInvalidCredentials):
		response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "认证失败")
	case errors.Is(err, service.ErrForbidden):
		response.Error(c, http.StatusForbidden, response.CodeForbidden, "没有操作权限")
	case errors.Is(err, service.ErrNotFound):
		response.Error(c, http.StatusNotFound, response.CodeNotFound, "账号或会话不存在")
	case errors.Is(err, service.ErrLastAdmin):
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "必须保留至少一个启用的管理员账号")
	case errors.Is(err, service.ErrUsernameTaken):
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "用户名已存在")
	case errors.Is(err, service.ErrWeakPassword):
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "密码需为 8～72 字节且不能全为空白")
	case errors.As(err, &validation):
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, validation.Message)
	default:
		response.Error(c, http.StatusInternalServerError, response.CodeError, "系统繁忙，请稍后再试")
	}
}

func (h *AuthHandler) ListSessions(c *gin.Context) {
	token, _ := c.Cookie(h.cfg.Auth.CookieName)
	sessions, err := h.service.ListSessions(c.Request.Context(), token)
	if err != nil {
		writeAccountError(c, err)
		return
	}
	response.OK(c, gin.H{"sessions": sessions})
}

func (h *AuthHandler) RevokeSession(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "会话 ID 无效")
		return
	}
	token, _ := c.Cookie(h.cfg.Auth.CookieName)
	current, err := h.service.RevokeSession(c.Request.Context(), token, id)
	status := service.AuditStatusSuccess
	if err != nil {
		status = service.AuditStatusFailure
	}
	recordAdminAudit(c, h.auditSvc, "auth.session_revoke", "session", id.String(), status, nil)
	if err != nil {
		writeAccountError(c, err)
		return
	}
	if current {
		middleware.ClearSessionCookie(c, h.cfg)
	}
	response.OK(c, gin.H{"current": current})
}

func (h *AuthHandler) ListAccounts(c *gin.Context) {
	token, _ := c.Cookie(h.cfg.Auth.CookieName)
	accounts, err := h.service.ListAccounts(c.Request.Context(), token)
	if err != nil {
		writeAccountError(c, err)
		return
	}
	response.OK(c, gin.H{"accounts": accounts})
}

func (h *AuthHandler) CreateAccount(c *gin.Context) {
	var req dto.CreateAccountReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "参数错误")
		return
	}
	token, _ := c.Cookie(h.cfg.Auth.CookieName)
	account, err := h.service.CreateAccount(c.Request.Context(), token, req)
	status := service.AuditStatusSuccess
	if err != nil {
		status = service.AuditStatusFailure
	}
	recordAdminAudit(c, h.auditSvc, "admin.create", "admin_account", account.ID, status, map[string]any{"username": req.Username, "role": req.Role})
	if err != nil {
		writeAccountError(c, err)
		return
	}
	response.OK(c, account)
}

func (h *AuthHandler) UpdateAccount(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "账号 ID 无效")
		return
	}
	var req dto.UpdateAccountReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "参数错误")
		return
	}
	token, _ := c.Cookie(h.cfg.Auth.CookieName)
	account, err := h.service.UpdateAccount(c.Request.Context(), token, id, req)
	status := service.AuditStatusSuccess
	if err != nil {
		status = service.AuditStatusFailure
	}
	// 只记录变更类型与非敏感字段，禁止把验密和新密码写入审计详情。
	detail := map[string]any{"username": req.Username, "role": req.Role, "disabled": req.Disabled, "password_reset": req.NewPassword != nil}
	recordAdminAudit(c, h.auditSvc, "admin.update", "admin_account", id.String(), status, detail)
	if err != nil {
		writeAccountError(c, err)
		return
	}
	response.OK(c, account)
}
