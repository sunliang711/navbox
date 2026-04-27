package dto

import "time"

type SiteListQuery struct {
	Search string
	TagIDs []string
	View   string
}

type SiteResp struct {
	ID              string    `json:"id"`
	Title           string    `json:"title"`
	Description     string    `json:"description"`
	DefaultURL      string    `json:"default_url"`
	LANURL          string    `json:"lan_url"`
	OpenMethod      string    `json:"open_method"`
	IconType        string    `json:"icon_type"`
	IconValue       string    `json:"icon_value"`
	BackgroundColor string    `json:"background_color"`
	OnlyName        bool      `json:"only_name"`
	IsFavorite      bool      `json:"is_favorite"`
	SortOrder       int       `json:"sort_order"`
	Tags            []TagResp `json:"tags"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type SiteSaveReq struct {
	Title           string   `json:"title" binding:"required,max=120"`
	Description     string   `json:"description"`
	DefaultURL      string   `json:"default_url" binding:"required"`
	LANURL          string   `json:"lan_url"`
	OpenMethod      string   `json:"open_method"`
	IconType        string   `json:"icon_type"`
	IconValue       string   `json:"icon_value"`
	BackgroundColor string   `json:"background_color"`
	OnlyName        bool     `json:"only_name"`
	IsFavorite      bool     `json:"is_favorite"`
	SortOrder       int      `json:"sort_order"`
	TagIDs          []string `json:"tag_ids"`
}

type BatchDeleteSitesReq struct {
	SiteIDs []string `json:"site_ids" binding:"required,min=1"`
}

type BatchSiteTagsReq struct {
	SiteIDs []string `json:"site_ids" binding:"required,min=1"`
	TagIDs  []string `json:"tag_ids" binding:"required,min=1"`
	Action  string   `json:"action" binding:"required,oneof=add remove"`
}

type OrderItemReq struct {
	ID        string `json:"id" binding:"required"`
	SortOrder int    `json:"sort_order"`
}

type UpdateOrderReq struct {
	Items []OrderItemReq `json:"items" binding:"required,min=1"`
}
