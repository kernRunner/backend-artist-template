package works

import (
	"registration-app/internal/domain/media"
	"time"
)

type ArtworkRevision struct {
	ID        string `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	ArtworkID string `gorm:"type:uuid;index;not null" json:"-"`

	ImageID *string      `gorm:"type:uuid" json:"image_id,omitempty"`
	Image   *media.Image `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"image,omitempty"`

	Year   string `json:"year,omitempty"`
	Medium string `json:"medium,omitempty"`
	SizeCM string `gorm:"column:size_cm" json:"size_cm,omitempty"`
	Price  string `json:"price,omitempty"`

	I18n []ArtworkI18nRevision `gorm:"constraint:OnDelete:CASCADE;" json:"i18n,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ArtworkI18nRevision struct {
	ArtworkRevisionID string `gorm:"type:uuid;primaryKey"`
	Lang              string `gorm:"primaryKey"`

	Title       string `gorm:"not null" json:"title"`
	Description string `json:"description,omitempty"`
	Notes       string `json:"notes,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
