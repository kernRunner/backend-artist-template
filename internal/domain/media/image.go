package media

import "time"

type Image struct {
	ID           string  `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	OriginalPath string  `gorm:"not null" json:"original_path"`
	WebpPath     *string `json:"webp_path,omitempty"`
	AvifPath     *string `json:"avif_path,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
