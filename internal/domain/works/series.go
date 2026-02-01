package works

import (
	"time"
)

const (
	OwnerUser   = "user"
	OwnerSystem = "system"
)

type Series struct {
	ID string `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`

	OwnerType string `gorm:"type:text;not null;default:'user';index" json:"-"`
	UserID    *uint  `gorm:"index" json:"-"`

	IDLocked bool `gorm:"not null;default:false" json:"id_locked"`

	PublishedRevisionID *string         `gorm:"type:uuid;index" json:"-"`
	DraftRevisionID     *string         `gorm:"type:uuid;index" json:"-"`
	DraftRevision       *SeriesRevision `gorm:"foreignKey:DraftRevisionID"`
	PublishedRevision   *SeriesRevision `gorm:"foreignKey:PublishedRevisionID"`

	Items []Artwork `gorm:"foreignKey:SeriesID;constraint:OnDelete:CASCADE;" json:"items,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
