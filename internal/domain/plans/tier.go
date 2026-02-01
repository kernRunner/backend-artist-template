package plans

import "strings"

// Tier constants (single source of truth)
const (
	TierNone         = "none"
	TierEssential    = "essential"
	TierProfessional = "professional"
	TierAdvanced     = "advanced"
)

// PlanTier returns the effective tier for a plan.
// Priority:
// 1. Explicit Tier stored in DB
// 2. Fallback inference by price (legacy safety net)
func PlanTier(p *Plan) string {
	if p == nil {
		return TierNone
	}

	// ✅ Prefer DB value
	tier := strings.ToLower(strings.TrimSpace(p.Tier))
	switch tier {
	case TierEssential, TierProfessional, TierAdvanced:
		return tier
	}

	// ⚠️ Fallback (should disappear over time)
	return inferTierFromPrice(p.PriceEUR)
}

// inferTierFromPrice exists ONLY as a backward-compatibility fallback.
// Do not rely on this long-term.
func inferTierFromPrice(priceEUR float64) string {
	switch {
	case priceEUR >= 320:
		return TierAdvanced
	case priceEUR >= 180:
		return TierProfessional
	default:
		return TierEssential
	}
}
