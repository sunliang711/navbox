package model

import "time"

// SystemSetting 存储系统级键值配置（例如自动生成的加密主密钥）。
type SystemSetting struct {
	Key       string    `json:"-" gorm:"column:key;type:varchar(128);primaryKey"`
	ValueText string    `json:"-" gorm:"column:value_text;type:text;not null;default:''"`
	UpdatedAt time.Time `json:"-" gorm:"column:updated_at;not null"`
}

func (SystemSetting) TableName() string {
	return "system_settings"
}
