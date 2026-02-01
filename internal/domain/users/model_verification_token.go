package users

import "time"

type VerificationToken struct {
	ID        uint   `gorm:"primaryKey"`
	UserID    uint   `gorm:"uniqueIndex"`
	User      User   `gorm:"constraint:OnDelete:CASCADE"`
	Token     string `gorm:"uniqueIndex"`
	Type      string `gorm:"index"`
	ExpiresAt time.Time
	CreatedAt time.Time
}
