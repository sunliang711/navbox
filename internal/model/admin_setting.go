package model

import "time"

type AdminSetting struct {
	ID           uint      `json:"-" gorm:"column:id;primaryKey"`
	PasswordHash string    `json:"-" gorm:"column:password_hash;type:text;not null;default:''"`
	Initialized  bool      `json:"-" gorm:"column:initialized;not null;default:false"`
	CreatedAt    time.Time `json:"-" gorm:"column:created_at"`
	UpdatedAt    time.Time `json:"-" gorm:"column:updated_at"`
}

func (AdminSetting) TableName() string {
	return "admin_settings"
}
