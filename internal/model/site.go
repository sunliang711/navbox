package model

type Site struct {
	Base
	Title           string `json:"-" gorm:"column:title;size:120;not null"`
	Description     string `json:"-" gorm:"column:description;type:text;not null;default:''"`
	DefaultURL      string `json:"-" gorm:"column:default_url;type:text;not null"`
	LANURL          string `json:"-" gorm:"column:lan_url;type:text;not null;default:''"`
	OpenMethod      string `json:"-" gorm:"column:open_method;size:32;not null;default:'new_window'"`
	IconType        string `json:"-" gorm:"column:icon_type;size:32;not null;default:'text'"`
	IconValue       string `json:"-" gorm:"column:icon_value;type:text;not null;default:''"`
	BackgroundColor string `json:"-" gorm:"column:background_color;size:32;not null;default:''"`
	OnlyName        bool   `json:"-" gorm:"column:only_name;not null;default:false"`
	IsFavorite      bool   `json:"-" gorm:"column:is_favorite;not null;default:false"`
	SortOrder       int    `json:"-" gorm:"column:sort_order;not null;default:0;index"`
}

func (Site) TableName() string {
	return "sites"
}
