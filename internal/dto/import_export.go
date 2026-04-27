package dto

import "time"

type ExportReq struct {
	SiteIDs []string `json:"site_ids"`
	TagIDs  []string `json:"tag_ids"`
}

type ExportArchiveResp struct {
	FileName string
	Content  []byte
}

type NavboxArchive struct {
	Version    int           `json:"version"`
	ExportedAt time.Time     `json:"exported_at"`
	Sites      []ArchiveSite `json:"sites"`
	Tags       []ArchiveTag  `json:"tags"`
	Icons      []ArchiveIcon `json:"icons"`
}

type ArchiveSite struct {
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
	TagIDs          []string  `json:"tag_ids"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type ArchiveTag struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Icon      string    `json:"icon"`
	Color     string    `json:"color"`
	SortOrder int       `json:"sort_order"`
	IsDefault bool      `json:"is_default"`
	IsEnabled bool      `json:"is_enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ArchiveIcon struct {
	ID        string    `json:"id"`
	FileName  string    `json:"file_name"`
	FilePath  string    `json:"file_path"`
	SHA256    string    `json:"sha256"`
	MIMEType  string    `json:"mime_type"`
	SizeBytes int64     `json:"size_bytes"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ImportReportResp struct {
	Imported  ImportCountResp      `json:"imported"`
	Skipped   ImportCountResp      `json:"skipped"`
	Conflicts []ImportConflictResp `json:"conflicts"`
}

type ImportCountResp struct {
	Sites     int `json:"sites"`
	Tags      int `json:"tags"`
	Icons     int `json:"icons"`
	Relations int `json:"relations"`
}

type ImportConflictResp struct {
	Type   string `json:"type"`
	ID     string `json:"id,omitempty"`
	Name   string `json:"name,omitempty"`
	Reason string `json:"reason"`
}
