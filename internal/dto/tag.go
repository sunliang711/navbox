package dto

import "time"

type TagResp struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Icon      string    `json:"icon"`
	Color     string    `json:"color"`
	SortOrder int       `json:"sort_order"`
	IsDefault bool      `json:"is_default"`
	IsEnabled bool      `json:"is_enabled"`
	SiteCount int64     `json:"site_count"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type TagSaveReq struct {
	Name      string `json:"name" binding:"required,max=80"`
	Icon      string `json:"icon"`
	Color     string `json:"color"`
	SortOrder int    `json:"sort_order"`
	IsEnabled *bool  `json:"is_enabled"`
}

type PublicConfigResp struct {
	DefaultTagID string `json:"default_tag_id"`
}
