package model

import "time"

// AuditLog 记录管理员操作与定时任务执行的结构化审计日志。
type AuditLog struct {
	ID           int64     `json:"-" gorm:"column:id;primaryKey;autoIncrement"`
	ActorType    string    `json:"-" gorm:"column:actor_type;type:varchar(32);not null"`
	ActorID      string    `json:"-" gorm:"column:actor_id;type:varchar(128);not null;default:''"`
	ActorName    string    `json:"-" gorm:"column:actor_name;type:varchar(128);not null;default:''"`
	Action       string    `json:"-" gorm:"column:action;type:varchar(128);not null;index:idx_audit_logs_action"`
	ResourceType string    `json:"-" gorm:"column:resource_type;type:varchar(64);not null;default:''"`
	ResourceID   string    `json:"-" gorm:"column:resource_id;type:varchar(128);not null;default:''"`
	Status       string    `json:"-" gorm:"column:status;type:varchar(32);not null;default:'success'"`
	IPAddress    string    `json:"-" gorm:"column:ip_address;type:varchar(64);not null;default:''"`
	UserAgent    string    `json:"-" gorm:"column:user_agent;type:text;not null;default:''"`
	RequestID    string    `json:"-" gorm:"column:request_id;type:varchar(64);not null;default:''"`
	DetailJSON   string    `json:"-" gorm:"column:detail_json;type:text;not null;default:'{}'"`
	CreatedAt    time.Time `json:"-" gorm:"column:created_at;not null;index:idx_audit_logs_created,sort:desc"`
}

func (AuditLog) TableName() string {
	return "audit_logs"
}
