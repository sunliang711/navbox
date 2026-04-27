package handler

import (
	"github.com/gin-gonic/gin"

	"navbox/internal/response"
)

type HealthHandler struct{}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

func (h *HealthHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/health", h.GetHealth)
}

func (h *HealthHandler) GetHealth(c *gin.Context) {
	response.OK(c, gin.H{"status": "ok"})
}
