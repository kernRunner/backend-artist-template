package access

import (
	"time"

	"registration-app/internal/domain/plans"
	"registration-app/internal/domain/users"
	"registration-app/internal/infra/stripe"
)

// Effective access for UI/product: trial|full|limited|locked
func ComputeEffectiveAccessState(now time.Time, u users.User) AccessState {
	// Active trial
	if u.TrialEndAt != nil && now.Before(*u.TrialEndAt) {
		return AccessTrial
	}

	// No subscription at all
	if u.SubscriptionId == nil || *u.SubscriptionId == "" {
		return AccessLocked
	}

	// Subscription exists -> interpret Stripe status
	switch stripe.NormalizeStripeStatus(u.StripeSubscriptionStatus) {
	case "active", "trialing":
		// Full vs limited decided by tier
		switch plans.PlanTier(u.Plan) {
		case "professional", "advanced":
			return AccessFull
		default:
			return AccessLimited
		}

	case "past_due":
		return AccessLimited

	case "canceled":
		// If you allow access until paid-through end date
		if u.CurrentPeriodEnd != nil && now.Before(*u.CurrentPeriodEnd) {
			switch plans.PlanTier(u.Plan) {
			case "professional", "advanced":
				return AccessFull
			default:
				return AccessLimited
			}
		}
		return AccessLocked

	default:
		return AccessLocked
	}
}
