package works

import (
	"registration-app/internal/domain/media"
	"time"
)

type SeriesRevision struct {
	ID       string `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	SeriesID string `gorm:"type:uuid;index;not null" json:"-"`

	ImageID *string      `gorm:"type:uuid" json:"image_id,omitempty"`
	Image   *media.Image `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"image,omitempty"`

	I18n []SeriesI18nRevision `gorm:"constraint:OnDelete:CASCADE;" json:"i18n,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type SeriesI18nRevision struct {
	SeriesRevisionID string `gorm:"type:uuid;primaryKey"`
	Lang             string `gorm:"primaryKey"`

	Title            string `gorm:"not null" json:"title"`
	DescriptionSerie string `json:"description_serie,omitempty"`
	Year             string `json:"year,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
