package model

type Icon struct {
	Base
	FileName  string `json:"-" gorm:"column:file_name;size:255;not null"`
	FilePath  string `json:"-" gorm:"column:file_path;type:text;not null"`
	SHA256    string `json:"-" gorm:"column:sha256;size:64;not null;uniqueIndex"`
	MIMEType  string `json:"-" gorm:"column:mime_type;size:120;not null"`
	SizeBytes int64  `json:"-" gorm:"column:size_bytes;not null"`
}

func (Icon) TableName() string {
	return "icons"
}
