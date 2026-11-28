package handler

import (
	"github.com/gin-gonic/gin"

	"navbox/internal/middleware"
	"navbox/internal/service"
)

func recordAdminAudit(
	c *gin.Context,
	auditSvc service.AuditService,
	action string,
	resourceType string,
	resourceID string,
	status string,
	detail any,
) {
	if auditSvc == nil || c == nil {
		return
	}
	actorID, actorName := service.AuditActorAdmin, service.AuditActorAdmin
	if account, ok := middleware.CurrentAccount(c); ok && account.ID != "" {
		actorID, actorName = account.ID, account.Username
	}
	auditSvc.Record(c.Request.Context(), service.AuditEvent{
		ActorType:    service.AuditActorAdmin,
		ActorID:      actorID,
		ActorName:    actorName,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Status:       status,
		IPAddress:    c.ClientIP(),
		UserAgent:    c.Request.UserAgent(),
		RequestID:    middleware.GetRequestID(c),
		Detail:       detail,
	})
}
