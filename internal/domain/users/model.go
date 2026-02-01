package users

import (
	"registration-app/internal/domain/plans"
	"time"
)

type User struct {
	ID           uint `gorm:"primaryKey"`
	Name         string
	Lastname     string
	Tel          string
	Email        string  `gorm:"not null;uniqueIndex:idx_users_email"`
	Password     *string `gorm:""`
	AuthProvider string  `gorm:"type:varchar(20);not null;default:'local'"`
	GoogleSub    *string `gorm:"uniqueIndex:idx_users_google_sub"`
	Role         string
	IsVerified   bool

	PlanID *uint
	Plan   *plans.Plan

	SubscriptionStart *time.Time
	SubscriptionEnd   *time.Time
	SubscriptionId    *string `gorm:"column:subscription_id;uniqueIndex:idx_users_subscription_id"`
	StripeCustomerID  *string `gorm:"column:stripe_customer_id;uniqueIndex:idx_users_stripe_customer_id"`

	// already in your model:
	PendingPlan          *plans.Plan `gorm:"foreignKey:PendingPlanID"`
	PendingPlanID        *uint       `gorm:"column:pending_plan_id"`
	PendingPlanStartDate *time.Time  `gorm:"column:pending_plan_start_date"`
	StripeScheduleID     *string     `gorm:"column:stripe_schedule_id"`
	CurrentPeriodEnd     *time.Time  `gorm:"column:current_period_end"`

	// ✅ NEW FIELDS (for /me output + Option C)
	TrialStartAt *time.Time `gorm:"column:trial_start_at"`
	TrialEndAt   *time.Time `gorm:"column:trial_end_at"`

	StripeSubscriptionStatus *string `gorm:"column:stripe_subscription_status"`

	SiteSlug *string `gorm:"column:site_slug;uniqueIndex:idx_users_site_slug"`

	CustomDomain *string `gorm:"column:custom_domain"`

	CreatedAt time.Time
	UpdatedAt time.Time
}
