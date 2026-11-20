package dto

import "time"

type AuditListQuery struct {
	ActorType string
	Action    string
	Status    string
	Query     string
	From      *time.Time
	To        *time.Time
	Limit     int
	Offset    int
}

type AuditLogResp struct {
	ID           int64     `json:"id"`
	ActorType    string    `json:"actor_type"`
	ActorID      string    `json:"actor_id"`
	ActorName    string    `json:"actor_name"`
	Action       string    `json:"action"`
	ResourceType string    `json:"resource_type"`
	ResourceID   string    `json:"resource_id"`
	Status       string    `json:"status"`
	IPAddress    string    `json:"ip_address"`
	UserAgent    string    `json:"user_agent"`
	RequestID    string    `json:"request_id"`
	DetailJSON   string    `json:"detail_json"`
	CreatedAt    time.Time `json:"created_at"`
}

type AuditListResp struct {
	Items []AuditLogResp `json:"items"`
	Total int64          `json:"total"`
}
