package plans

type Plan struct {
	ID            uint `gorm:"primaryKey"`
	Name          string
	PriceEUR      float64
	StripePriceID string `gorm:"column:stripe_price_id;not null;uniqueIndex:idx_plans_stripe_price_id"`
	Interval      string
	Tier          string `gorm:"column:tier"` // "essential" | "professional" | "advanced"

}
