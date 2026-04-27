package model

type Tag struct {
	Base
	Name      string `json:"-" gorm:"column:name;size:80;not null;uniqueIndex"`
	Icon      string `json:"-" gorm:"column:icon;type:text;not null;default:''"`
	Color     string `json:"-" gorm:"column:color;size:32;not null;default:''"`
	SortOrder int    `json:"-" gorm:"column:sort_order;not null;default:0;index"`
	IsDefault bool   `json:"-" gorm:"column:is_default;not null;default:false;index"`
	IsEnabled bool   `json:"-" gorm:"column:is_enabled;not null;default:true;index"`
}

func (Tag) TableName() string {
	return "tags"
}
