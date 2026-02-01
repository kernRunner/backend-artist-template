package access

import (
	"registration-app/internal/domain/plans"
)

func CapabilitiesFor(state AccessState, plan *plans.Plan) []string {
	// locked/limited: no edit capabilities
	if state == AccessLocked || state == AccessLimited {
		return []string{}
	}

	// trial
	if state == AccessTrial {
		return []string{"edit", "upload"}
	}

	// full: tier-based
	switch plans.PlanTier(plan) {
	case "essential":
		return []string{"edit", "upload"}
	case "professional":
		return []string{"edit", "upload", "custom_domain"}
	case "advanced":
		return []string{"edit", "upload", "custom_domain", "advanced_features"}
	default:
		return []string{"edit", "upload"}
	}
}
