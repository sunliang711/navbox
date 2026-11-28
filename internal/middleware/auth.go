package middleware

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"navbox/internal/config"
	"navbox/internal/dto"
	"navbox/internal/model"
	"navbox/internal/response"
	"navbox/internal/service"
)

const accountContextKey = "authenticated_account"

func SetCurrentAccount(c *gin.Context, account dto.AccountResp) {
	c.Set(accountContextKey, account)
}

func CurrentAccount(c *gin.Context) (dto.AccountResp, bool) {
	value, exists := c.Get(accountContextKey)
	account, ok := value.(dto.AccountResp)
	return account, exists && ok
}

// AdminPermissions 允许只读账号查询普通管理数据，写操作和凭据查看仅限管理员。
func AdminPermissions() gin.HandlerFunc {
	return func(c *gin.Context) {
		account, ok := CurrentAccount(c)
		if !ok || (account.Role != model.RoleAdmin &&
			(c.Request.Method != http.MethodGet || strings.HasSuffix(c.FullPath(), "/secrets"))) {
			response.Error(c, http.StatusForbidden, response.CodeForbidden, "没有操作权限")
			c.Abort()
			return
		}
		c.Next()
	}
}

func AdminSession(cfg config.Config, authService service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		SetNoCacheHeaders(c)

		token, err := c.Cookie(cfg.Auth.CookieName)
		if err != nil {
			response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "认证失败")
			c.Abort()
			return
		}
		result, err := authService.ValidateSessionWithRenewal(c.Request.Context(), token)
		if err != nil {
			if errors.Is(err, service.ErrUnauthenticated) {
				response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "认证失败")
				c.Abort()
				return
			}
			response.Error(c, http.StatusInternalServerError, response.CodeError, "系统繁忙，请稍后再试")
			c.Abort()
			return
		}
		if result.Renewed {
			SetSessionCookie(c, cfg, token)
		}
		SetCurrentAccount(c, result.User)
		c.Next()
	}
}

// TouchAdminSession 在公开接口上对已持有管理员 Session Cookie 的请求执行静默滑动续期，
// 避免管理员长期停留在游客首页时 Session 过期；无 Cookie 的普通游客直接放行，不产生任何数据库查询或写入。
func TouchAdminSession(cfg config.Config, authService service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie(cfg.Auth.CookieName)
		if err == nil && token != "" {
			if result, vErr := authService.ValidateSessionWithRenewal(c.Request.Context(), token); vErr == nil && result.Renewed {
				c.Header("Vary", "Cookie")
				SetSessionCookie(c, cfg, token)
			}
		}
		c.Next()
	}
}

func SetNoCacheHeaders(c *gin.Context) {
	c.Header("Cache-Control", "no-store, private")
	c.Header("Pragma", "no-cache")
	c.Header("Vary", "Cookie")
}

func SetSessionCookie(c *gin.Context, cfg config.Config, token string) {
	ttl := sessionDuration(cfg)
	cookie := &http.Cookie{
		Name:     cfg.Auth.CookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   cfg.Auth.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	}
	if ttl > 0 {
		cookie.MaxAge = int(ttl.Seconds())
		cookie.Expires = time.Now().UTC().Add(ttl)
	}
	http.SetCookie(c.Writer, cookie)
}

func ClearSessionCookie(c *gin.Context, cfg config.Config) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     cfg.Auth.CookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		Expires:  time.Unix(0, 0).UTC(),
		HttpOnly: true,
		Secure:   cfg.Auth.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

func sessionDuration(cfg config.Config) time.Duration {
	ttl, err := time.ParseDuration(cfg.Auth.SessionTTL)
	if err != nil || ttl <= 0 {
		return 0
	}
	return ttl
}
