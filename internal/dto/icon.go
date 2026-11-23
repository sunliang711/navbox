package dto

import "time"

type IconResp struct {
	ID        string    `json:"id"`
	FileName  string    `json:"file_name"`
	FilePath  string    `json:"file_path"`
	URL       string    `json:"url"`
	SHA256    string    `json:"sha256"`
	MIMEType  string    `json:"mime_type"`
	SizeBytes int64     `json:"size_bytes"`
	CreatedAt time.Time `json:"created_at"`
}

type FetchIconReq struct {
	URL string `json:"url" binding:"required"`
}

type IconSiteRef struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

type IconListItem struct {
	IconResp
	UsedBy []IconSiteRef `json:"used_by"`
}

type IconListQuery struct {
	Q      string
	Unused bool
	Limit  int
	Offset int
}

type IconListResp struct {
	Items []IconListItem `json:"items"`
	Total int            `json:"total"`
}
