package works

import "time"

type SeriesI18n struct {
	SeriesID         string `gorm:"type:uuid;primaryKey"`
	Lang             string `gorm:"primaryKey"`
	Title            string `gorm:"not null" json:"title"`
	DescriptionSerie string `json:"description_serie,omitempty"`
	Year             string `json:"year,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
