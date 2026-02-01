package site

import (
	"encoding/json"
	"time"
)

const (
	OwnerUser   = "user"
	OwnerSystem = "system"
)

type Template struct {
	ID     string `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Slug   string `gorm:"not null;uniqueIndex" json:"slug"`
	Name   string `gorm:"not null" json:"name"`
	Active bool   `gorm:"not null;default:true" json:"active"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type SitePage struct {
	ID string `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`

	OwnerType string `gorm:"not null;index" json:"owner_type"`
	UserID    *uint  `gorm:"index" json:"-"`

	TemplateID *string `gorm:"type:uuid;index" json:"template_id,omitempty"`

	Slug   string `gorm:"not null;index" json:"slug"`
	Lang   string `gorm:"not null;index" json:"lang"`
	Status string `gorm:"not null;default:'draft'" json:"status"`

	Blocks []SitePageBlock `gorm:"foreignKey:PageID;references:ID;constraint:OnDelete:CASCADE;" json:"blocks,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type SitePageBlock struct {
	ID string `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`

	PageID    string `gorm:"type:uuid;not null;index" json:"page_id"`
	SortIndex int    `gorm:"not null;default:0;index" json:"sort_index"`

	Type  string          `gorm:"not null;index" json:"type"`
	Props json.RawMessage `gorm:"type:jsonb;not null;default:'{}'" json:"props"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
