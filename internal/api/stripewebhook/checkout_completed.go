package stripewebhooks

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"registration-app/database"
	"registration-app/internal/domain/plans"
	"registration-app/internal/domain/users"

	"github.com/gin-gonic/gin"
	"github.com/stripe/stripe-go/v75"
	checkoutsession "github.com/stripe/stripe-go/v75/checkout/session"
	"github.com/stripe/stripe-go/v75/subscription"
)

func handleCheckoutSessionCompleted(c *gin.Context, session *stripe.CheckoutSession) error {
	// Fetch full session with expansions
	fullSession, err := checkoutsession.Get(session.ID, &stripe.CheckoutSessionParams{
		Params: stripe.Params{
			Expand: []*string{
				stripe.String("subscription"),
				stripe.String("customer"),
			},
		},
	})
	if err != nil {
		return fmt.Errorf("failed to fetch expanded checkout session: %w", err)
	}

	if fullSession.Subscription == nil || fullSession.Subscription.ID == "" {
		return errors.New("checkout session missing subscription")
	}
	subscriptionID := fullSession.Subscription.ID

	subData, err := subscription.Get(subscriptionID, nil)
	if err != nil || subData == nil || subData.Items == nil || len(subData.Items.Data) == 0 || subData.Items.Data[0].Price == nil {
		return fmt.Errorf("failed to fetch subscription items: %w", err)
	}

	// Identify user: metadata.user_id preferred, else ClientReferenceID
	userID, err := userIDFromSubscriptionOrRef(subData, fullSession.ClientReferenceID)
	if err != nil {
		return err
	}

	var user users.User
	if err := database.DB.Where("id = ?", userID).First(&user).Error; err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Map Stripe price -> Plan
	priceID := subData.Items.Data[0].Price.ID
	var plan plans.Plan
	if err := database.DB.Where("stripe_price_id = ?", priceID).First(&plan).Error; err != nil {
		return fmt.Errorf("plan not found for stripe price_id=%s: %w", priceID, err)
	}

	now := time.Now()
	periodEnd := time.Unix(subData.CurrentPeriodEnd, 0)
	status := string(subData.Status)

	updates := map[string]interface{}{
		"plan_id":                    plan.ID,
		"subscription_id":            subscriptionID,
		"subscription_start":         now,
		"subscription_end":           periodEnd,
		"current_period_end":         periodEnd,
		"stripe_subscription_status": status,
		"trial_start_at":             nil,
		"trial_end_at":               nil,
		"pending_plan_id":            nil,
		"pending_plan_start_date":    nil,
		"stripe_schedule_id":         nil,
	}

	if fullSession.Customer != nil && fullSession.Customer.ID != "" {
		updates["stripe_customer_id"] = fullSession.Customer.ID
	}

	// Optional: cancel old sub if different (be careful—can surprise users if multi-subscriptions)
	if user.SubscriptionId != nil && *user.SubscriptionId != "" && *user.SubscriptionId != subscriptionID {
		_, _ = subscription.Cancel(*user.SubscriptionId, nil)
	}

	if err := database.DB.Model(&users.User{}).
		Where("id = ?", user.ID).
		Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update user after checkout: %w", err)
	}

	return nil
}

func userIDFromSubscriptionOrRef(sub *stripe.Subscription, clientRef string) (uint, error) {
	userIDStr := ""
	if sub.Metadata != nil {
		userIDStr = sub.Metadata["user_id"]
	}
	if userIDStr == "" {
		userIDStr = clientRef
	}
	if userIDStr == "" {
		return 0, errors.New("missing user_id (metadata.user_id or client_reference_id)")
	}

	uid64, err := strconv.ParseUint(userIDStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid user_id %q: %w", userIDStr, err)
	}
	return uint(uid64), nil
}
