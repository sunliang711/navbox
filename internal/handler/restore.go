package handler

import (
	"context"
	"crypto/subtle"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"

	"navbox/internal/config"
	"navbox/internal/dto"
	"navbox/internal/response"
	"navbox/internal/service"
)

const restoreTokenTTL = 5 * time.Minute

type RestorePasswordService interface {
	ResetPassword(ctx context.Context, newPassword string) error
}

type RestoreHandler struct {
	service        RestorePasswordService
	cfg            config.Config
	logger         zerolog.Logger
	tokenMu        sync.Mutex
	tokenUsed      bool
	tokenExpiresAt time.Time
}

func NewRestoreHandler(cfg config.Config, logger zerolog.Logger, authService service.AuthService) *RestoreHandler {
	return &RestoreHandler{
		service:        authService,
		cfg:            cfg,
		logger:         logger.With().Str("component", "restore").Logger(),
		tokenExpiresAt: time.Now().Add(restoreTokenTTL),
	}
}

func (h *RestoreHandler) RegisterRoutes(rg *gin.RouterGroup) {
	restore := rg.Group("/restore")
	restore.GET("/status", h.GetStatus)
	restore.POST("/admin-password", h.ResetAdminPassword)
}

func (h *RestoreHandler) GetStatus(c *gin.Context) {
	response.OK(c, dto.RestoreStatusResp{
		Enabled: h.cfg.Restore.Enabled(),
		Mode:    h.cfg.Restore.Mode,
	})
}

func (h *RestoreHandler) ResetAdminPassword(c *gin.Context) {
	var req dto.RestoreAdminPasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "参数错误")
		return
	}

	if !h.consumeRestoreToken(req.RestoreToken) {
		h.logger.Warn().Str("remote_ip", c.ClientIP()).Msg("restore token rejected")
		response.Error(c, http.StatusForbidden, response.CodeForbidden, "Restore Token 无效、已过期或已使用")
		return
	}

	if err := h.service.ResetPassword(c.Request.Context(), req.NewPassword); err != nil {
		h.releaseRestoreToken()
		if errors.Is(err, service.ErrWeakPassword) {
			response.Error(c, http.StatusBadRequest, response.CodeBadRequest, "新密码强度不足")
			return
		}
		response.Error(c, http.StatusInternalServerError, response.CodeError, "系统繁忙，请稍后再试")
		return
	}

	h.logger.Info().Str("remote_ip", c.ClientIP()).Msg("admin password restored")
	response.OK(c, dto.RestoreAdminPasswordResp{Restored: true})
}

func (h *RestoreHandler) consumeRestoreToken(token string) bool {
	h.tokenMu.Lock()
	defer h.tokenMu.Unlock()

	if h.tokenUsed {
		return false
	}
	if time.Now().After(h.tokenExpiresAt) {
		return false
	}

	expectedToken := strings.TrimSpace(h.cfg.Restore.Token)
	actualToken := strings.TrimSpace(token)
	if expectedToken == "" || len(expectedToken) != len(actualToken) {
		return false
	}
	if subtle.ConstantTimeCompare([]byte(expectedToken), []byte(actualToken)) != 1 {
		return false
	}

	h.tokenUsed = true
	return true
}

func (h *RestoreHandler) releaseRestoreToken() {
	h.tokenMu.Lock()
	defer h.tokenMu.Unlock()

	h.tokenUsed = false
}
