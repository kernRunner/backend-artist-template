package users

import (
	"time"

	"registration-app/internal/domain/access"
	"registration-app/internal/domain/plans"
	"registration-app/internal/domain/site"
	"registration-app/internal/domain/users"
	"registration-app/internal/infra/stripe"
)

func BuildPlanDTO(p *plans.Plan) *PlanDTO {
	if p == nil {
		return nil
	}
	return &PlanDTO{
		ID:            p.ID,
		Key:           p.Name,
		Interval:      p.Interval,
		PriceEUR:      p.PriceEUR,
		StripePriceID: p.StripePriceID,
	}
}

func BuildSubscriptionDTO(u users.User) *SubscriptionDTO {
	if u.SubscriptionId == nil || *u.SubscriptionId == "" {
		return nil
	}
	return &SubscriptionDTO{
		Status:               stripe.NormalizeStripeStatus(u.StripeSubscriptionStatus),
		StartsAt:             u.SubscriptionStart,
		CurrentPeriodEnd:     u.CurrentPeriodEnd,
		StripeSubscriptionID: u.SubscriptionId,
		StripeScheduleID:     u.StripeScheduleID,
	}
}

func BuildTrialDTO(now time.Time, start, end *time.Time) *TrialDTO {
	if start == nil || end == nil {
		return nil
	}

	var daysLeft *int
	if now.Before(*end) {
		d := int(time.Until(*end).Hours() / 24)
		if d < 0 {
			d = 0
		}
		daysLeft = &d
	} else {
		d := 0
		daysLeft = &d
	}

	return &TrialDTO{
		StartsAt: start,
		EndsAt:   end,
		DaysLeft: daysLeft,
	}
}

func BuildPendingChangeDTO(u users.User) *PendingChangeDTO {
	if u.PendingPlanID == nil || u.PendingPlan == nil || u.PendingPlanStartDate == nil {
		return nil
	}
	return &PendingChangeDTO{
		EffectiveAt: u.PendingPlanStartDate,
		Plan: &PlanLiteDTO{
			Key:      u.PendingPlan.Name,
			Interval: u.PendingPlan.Interval,
			PriceEUR: u.PendingPlan.PriceEUR,
		},
	}
}

func BuildAccessSiteDTO(user users.User, policy access.Policy, limits *LimitsDTO) AccessSiteDTO {
	platformURL := ""
	slug := ""

	if user.SiteSlug != nil && *user.SiteSlug != "" {
		platformURL = site.BuildPublicURL(*user.SiteSlug)
		slug = *user.SiteSlug
	}

	var customDomain *string
	if policy.State != access.AccessLocked {
		tier := plans.PlanTier(user.Plan)
		if tier == "professional" || tier == "advanced" {
			customDomain = user.CustomDomain
		}
	}

	return AccessSiteDTO{
		Mode:         string(policy.EditorMode),
		PlatformURL:  platformURL,
		Slug:         slug,
		CustomDomain: customDomain,
		Limits:       limits,
	}
}
