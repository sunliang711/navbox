package model

import "time"

const (
	RoleAdmin  = "admin"
	RoleViewer = "viewer"
)

type AdminAccount struct {
	Base
	Username     string     `json:"-" gorm:"column:username;type:varchar(64);not null;uniqueIndex"`
	PasswordHash string     `json:"-" gorm:"column:password_hash;type:text;not null"`
	Role         string     `json:"-" gorm:"column:role;type:text;not null"`
	Disabled     bool       `json:"-" gorm:"column:disabled;not null;default:false"`
	IsPrimary    bool       `json:"-" gorm:"column:is_primary;not null;default:false"`
	LastLoginAt  *time.Time `json:"-" gorm:"column:last_login_at"`
}

func (AdminAccount) TableName() string {
	return "admin_accounts"
}
