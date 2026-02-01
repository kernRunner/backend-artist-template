package works

import (
	"registration-app/internal/domain/media"
	"time"
)

type Artwork struct {
	ID string `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`

	OwnerType string `gorm:"type:text;not null;default:'user';index" json:"-"`
	UserID    *uint  `gorm:"index" json:"-"`

	SortIndex int    `gorm:"not null;default:0;index:idx_artworks_series_sort,priority:2"`
	SeriesID  string `gorm:"type:uuid;not null;index:idx_artworks_series_sort,priority:1"`

	IDLocked bool `gorm:"not null;default:false" json:"id_locked"`
	Sold     bool `gorm:"not null;default:false" json:"sold"`

	ImageID *string      `gorm:"type:uuid" json:"image_id,omitempty"`
	Image   *media.Image `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"image,omitempty"`

	Year   string `json:"year,omitempty"`
	Medium string `json:"medium,omitempty"`
	SizeCM string `gorm:"column:size_cm" json:"size_cm,omitempty"`
	Price  string `json:"price,omitempty"`

	I18n []ArtworkI18n `gorm:"constraint:OnDelete:CASCADE;" json:"i18n,omitempty"`

	PublishedRevisionID *string          `gorm:"type:uuid;index" json:"-"`
	DraftRevisionID     *string          `gorm:"type:uuid;index" json:"-"`
	DraftRevision       *ArtworkRevision `gorm:"foreignKey:DraftRevisionID"`
	PublishedRevision   *ArtworkRevision `gorm:"foreignKey:PublishedRevisionID"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
