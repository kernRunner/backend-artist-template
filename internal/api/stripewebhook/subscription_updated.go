package stripewebhooks

import (
	"fmt"
	"strconv"
	"time"

	"registration-app/database"
	"registration-app/internal/domain/plans"
	"registration-app/internal/domain/users"

	"github.com/gin-gonic/gin"
	"github.com/stripe/stripe-go/v75"
)

func handleSubscriptionUpdated(c *gin.Context, sub *stripe.Subscription) error {
	if sub.ID == "" || sub.Items == nil || len(sub.Items.Data) == 0 || sub.Items.Data[0].Price == nil {
		return fmt.Errorf("subscription missing id/items/price")
	}

	subscriptionID := sub.ID
	activePriceID := sub.Items.Data[0].Price.ID
	periodEnd := time.Unix(sub.CurrentPeriodEnd, 0)
	status := string(sub.Status)

	// Find user
	var user users.User
	userID := userIDFromMetadata(sub.Metadata)
	if userID != 0 {
		if err := database.DB.Where("id = ?", userID).First(&user).Error; err != nil {
			// acknowledge to avoid Stripe retries if user deleted
			return nil
		}
	} else {
		if err := database.DB.Where("subscription_id = ?", subscriptionID).First(&user).Error; err != nil {
			return nil
		}
	}

	// Map plan
	var plan plans.Plan
	if err := database.DB.Where("stripe_price_id = ?", activePriceID).First(&plan).Error; err != nil {
		return nil
	}

	updates := map[string]interface{}{
		"plan_id":                    plan.ID,
		"subscription_end":           periodEnd,
		"current_period_end":         periodEnd,
		"stripe_subscription_status": status,
		"subscription_id":            subscriptionID,
	}

	// If pending downgrade matches current plan -> clear pending
	if user.PendingPlanID != nil && *user.PendingPlanID == plan.ID {
		updates["pending_plan_id"] = nil
		updates["pending_plan_start_date"] = nil
		updates["stripe_schedule_id"] = nil
	}

	return database.DB.Model(&users.User{}).
		Where("id = ?", user.ID).
		Updates(updates).Error
}

func userIDFromMetadata(md map[string]string) uint {
	if md == nil {
		return 0
	}
	s := md["user_id"]
	if s == "" {
		return 0
	}
	uid, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0
	}
	return uint(uid)
}
