package model

import (
	"time"

	"github.com/google/uuid"
)

type SiteTag struct {
	SiteID    uuid.UUID `json:"-" gorm:"column:site_id;type:uuid;primaryKey"`
	TagID     uuid.UUID `json:"-" gorm:"column:tag_id;type:uuid;primaryKey"`
	CreatedAt time.Time `json:"-" gorm:"column:created_at"`
}

func (SiteTag) TableName() string {
	return "site_tags"
}
