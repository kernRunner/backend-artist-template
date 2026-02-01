package billing

import (
	"registration-app/internal/domain/plans"
	"registration-app/internal/domain/users"
	"time"
)

type Payment struct {
	ID                   uint `gorm:"primaryKey"`
	UserID               uint
	User                 users.User
	PlanID               *uint
	Plan                 *plans.Plan
	StripeSessionID      string `gorm:"uniqueIndex"`
	StripeSubscriptionID *string
	AmountEUR            float64
	Status               string
	InvoiceID            *string
	ReceiptURL           *string
	CreatedAt            time.Time
}
