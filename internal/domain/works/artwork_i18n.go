package works

import "time"

type ArtworkI18n struct {
	ArtworkID   string `gorm:"type:uuid;primaryKey"`
	Lang        string `gorm:"primaryKey"`
	Title       string `gorm:"not null" json:"title"`
	Description string `json:"description,omitempty"`
	Notes       string `json:"notes,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
