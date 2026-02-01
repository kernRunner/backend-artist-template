package users

import "time"

type MeResponse struct {
	User    UserDTO    `json:"user"`
	Billing BillingDTO `json:"billing"`
	Access  AccessDTO  `json:"access"`
}

/* ---------- USER ---------- */

type UserDTO struct {
	ID         uint    `json:"id"`
	Email      string  `json:"email"`
	Name       string  `json:"name"`
	Lastname   string  `json:"lastname"`
	Tel        *string `json:"tel"`
	Role       string  `json:"role"`
	IsVerified bool    `json:"is_verified"`
}

/* ---------- BILLING ---------- */

type BillingDTO struct {
	Plan          *PlanDTO          `json:"plan"`
	Subscription  *SubscriptionDTO  `json:"subscription"`
	Trial         *TrialDTO         `json:"trial"`
	PendingChange *PendingChangeDTO `json:"pending_change"`
}

type PlanDTO struct {
	ID            uint    `json:"id"`
	Key           string  `json:"key"`
	Interval      string  `json:"interval"`
	PriceEUR      float64 `json:"price_eur"`
	StripePriceID string  `json:"stripe_price_id"`
}

type SubscriptionDTO struct {
	Status               string     `json:"status"`
	StartsAt             *time.Time `json:"starts_at"`
	CurrentPeriodEnd     *time.Time `json:"current_period_end"`
	StripeSubscriptionID *string    `json:"stripe_subscription_id"`
	StripeScheduleID     *string    `json:"stripe_schedule_id"`
}

type TrialDTO struct {
	StartsAt *time.Time `json:"starts_at"`
	EndsAt   *time.Time `json:"ends_at"`
	DaysLeft *int       `json:"days_left"`
}

type PendingChangeDTO struct {
	EffectiveAt *time.Time   `json:"effective_at"`
	Plan        *PlanLiteDTO `json:"plan"`
}

type PlanLiteDTO struct {
	Key      string  `json:"key"`
	Interval string  `json:"interval"`
	PriceEUR float64 `json:"price_eur"`
}

/* ---------- ACCESS ---------- */

type AccessDTO struct {
	State        string        `json:"state"` // trial|full|limited|locked
	Capabilities []string      `json:"capabilities"`
	Site         AccessSiteDTO `json:"site"`
}

type AccessSiteDTO struct {
	Mode         string     `json:"mode"` // full|limited
	PlatformURL  string     `json:"platform_url"`
	Slug         string     `json:"slug"`
	CustomDomain *string    `json:"custom_domain"`
	Limits       *LimitsDTO `json:"limits,omitempty"`
}

type LimitsDTO struct {
	MaxArtworks          int  `json:"max_artworks"`
	HideGalleries        bool `json:"hide_galleries"`
	NoIndex              bool `json:"noindex"`
	ShowPlatformBranding bool `json:"show_platform_branding"`
}
