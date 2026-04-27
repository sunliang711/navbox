package handler

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"navbox/internal/config"
	"navbox/internal/dto"
	"navbox/internal/response"
	"navbox/internal/service"
)

type AuthHandler struct {
	cfg     config.Config
	service service.AuthService
}

func NewAuthHandler(cfg config.Config, service service.AuthService) *AuthHandler {
	return &AuthHandler{cfg: cfg, service: service}
}

func (h *AuthHandler) RegisterRoutes(rg *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	admin := rg.Group("/admin")
	admin.POST("/login", h.Login)
	admin.POST("/logout", authMiddleware, h.Logout)
	admin.GET("/session", authMiddleware, h.GetSession)
	admin.POST("/password", authMiddleware, h.ChangePassword)
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req dto.LoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "参数错误")
		return
	}

	result, err := h.service.Login(c.Request.Context(), req.Password)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "认证失败")
			return
		}
		response.Error(c, http.StatusInternalServerError, response.CodeError, "系统繁忙，请稍后再试")
		return
	}

	h.setSessionCookie(c, result.Token)
	response.OK(c, dto.SessionResp{Authenticated: true})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	token, _ := c.Cookie(h.cfg.Auth.CookieName)
	if err := h.service.Logout(c.Request.Context(), token); err != nil {
		response.Error(c, http.StatusInternalServerError, response.CodeError, "系统繁忙，请稍后再试")
		return
	}

	h.clearSessionCookie(c)
	response.OK(c, dto.SessionResp{Authenticated: false})
}

func (h *AuthHandler) GetSession(c *gin.Context) {
	response.OK(c, dto.SessionResp{Authenticated: true})
}

func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req dto.ChangePasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "参数错误")
		return
	}

	token, _ := c.Cookie(h.cfg.Auth.CookieName)
	if err := h.service.ChangePassword(c.Request.Context(), token, req.CurrentPassword, req.NewPassword); err != nil {
		switch {
		case errors.Is(err, service.ErrUnauthenticated):
			response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "认证失败")
		case errors.Is(err, service.ErrInvalidCredentials):
			response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "认证失败")
		case errors.Is(err, service.ErrWeakPassword):
			response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "新密码强度不足")
		default:
			response.Error(c, http.StatusInternalServerError, response.CodeError, "系统繁忙，请稍后再试")
		}
		return
	}

	response.OK(c, dto.SessionResp{Authenticated: true})
}

func (h *AuthHandler) setSessionCookie(c *gin.Context, token string) {
	maxAge := h.sessionMaxAge()
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(h.cfg.Auth.CookieName, token, maxAge, "/", "", h.cfg.Auth.CookieSecure, true)
}

func (h *AuthHandler) clearSessionCookie(c *gin.Context) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(h.cfg.Auth.CookieName, "", -1, "/", "", h.cfg.Auth.CookieSecure, true)
}

func (h *AuthHandler) sessionMaxAge() int {
	ttl, err := time.ParseDuration(h.cfg.Auth.SessionTTL)
	if err != nil {
		return 0
	}
	return int(ttl.Seconds())
}
