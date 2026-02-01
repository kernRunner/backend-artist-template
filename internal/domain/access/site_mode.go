package access

import "registration-app/internal/domain/plans"

func EditorModeFromState(state AccessState) EditorMode {
	if state == AccessFull || state == AccessTrial {
		return EditorFull
	}
	return EditorLimited
}

func PublicModeFromState(state AccessState, plan *plans.Plan) PublicMode {
	if state == AccessLimited || state == AccessLocked {
		return PublicLimited
	}
	if state == AccessTrial {
		return PublicFull
	}

	switch plans.PlanTier(plan) { // or plans.Tier(plan) if you renamed it
	case "professional", "advanced":
		return PublicFull
	default:
		return PublicLimited
	}
}
