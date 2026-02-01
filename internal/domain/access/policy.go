package access

import (
	"time"

	"registration-app/internal/domain/site"
	"registration-app/internal/domain/users"
)

type Policy struct {
	State        AccessState
	EditorMode   EditorMode
	Capabilities []string
	Limits       *site.LimitedRules
}

func ComputePolicy(now time.Time, u users.User) Policy {
	state := ComputeEffectiveAccessState(now, u)

	return Policy{
		State:        state,
		EditorMode:   EditorModeFromState(state),
		Capabilities: CapabilitiesFor(state, u.Plan),
		Limits:       site.LimitedRulesFor(string(state)),
	}
}
