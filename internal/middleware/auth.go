package middleware

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"navbox/internal/config"
	"navbox/internal/response"
	"navbox/internal/service"
)

func AdminSession(cfg config.Config, authService service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
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
			setSessionCookie(c, cfg, token)
		}
		c.Next()
	}
}

func setSessionCookie(c *gin.Context, cfg config.Config, token string) {
	maxAge := sessionMaxAge(cfg)
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(cfg.Auth.CookieName, token, maxAge, "/", "", cfg.Auth.CookieSecure, true)
}

func sessionMaxAge(cfg config.Config) int {
	ttl, err := time.ParseDuration(cfg.Auth.SessionTTL)
	if err != nil {
		return 0
	}
	return int(ttl.Seconds())
}
