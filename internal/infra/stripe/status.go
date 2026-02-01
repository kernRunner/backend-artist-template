package stripe

import "strings"

// Stripe-ish normalization used ONLY for billing.subscription.status
func NormalizeStripeStatus(s *string) string {
	if s == nil || strings.TrimSpace(*s) == "" {
		return "none"
	}
	switch strings.TrimSpace(*s) {
	case "active":
		return "active"
	case "trialing":
		return "trialing"
	case "past_due", "unpaid":
		return "past_due"
	case "canceled", "incomplete_expired":
		return "canceled"
	default:
		return strings.TrimSpace(*s)
	}
}
