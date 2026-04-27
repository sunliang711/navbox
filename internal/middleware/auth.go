package middleware

import (
	"errors"
	"net/http"

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
		if err := authService.ValidateSession(c.Request.Context(), token); err != nil {
			if errors.Is(err, service.ErrUnauthenticated) {
				response.Error(c, http.StatusUnauthorized, response.CodeUnauthorized, "认证失败")
				c.Abort()
				return
			}
			response.Error(c, http.StatusInternalServerError, response.CodeError, "系统繁忙，请稍后再试")
			c.Abort()
			return
		}
		c.Next()
	}
}
