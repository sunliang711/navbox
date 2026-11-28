package model

import (
	"time"

	"github.com/google/uuid"
)

type AdminSession struct {
	Base
	AccountID  *uuid.UUID `json:"-" gorm:"column:account_id;type:uuid;index"`
	ClientIP   string     `json:"-" gorm:"column:client_ip;type:text;not null;default:''"`
	UserAgent  string     `json:"-" gorm:"column:user_agent;type:text;not null;default:''"`
	LastSeenAt *time.Time `json:"-" gorm:"column:last_seen_at"`
	TokenHash  string     `json:"-" gorm:"column:token_hash;type:text;not null;uniqueIndex"`
	ExpiresAt  time.Time  `json:"-" gorm:"column:expires_at;not null;index"`
}

func (AdminSession) TableName() string {
	return "admin_sessions"
}
