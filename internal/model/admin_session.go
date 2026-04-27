package model

import "time"

type AdminSession struct {
	Base
	TokenHash string    `json:"-" gorm:"column:token_hash;type:text;not null;uniqueIndex"`
	ExpiresAt time.Time `json:"-" gorm:"column:expires_at;not null;index"`
}

func (AdminSession) TableName() string {
	return "admin_sessions"
}
