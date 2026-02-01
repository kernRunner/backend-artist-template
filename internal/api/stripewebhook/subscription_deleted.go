package stripewebhooks

import (
	"time"

	"registration-app/database"
	"registration-app/internal/domain/users"

	"github.com/gin-gonic/gin"
	"github.com/stripe/stripe-go/v75"
)

func handleSubscriptionDeleted(c *gin.Context, sub *stripe.Subscription) error {
	if sub.ID == "" {
		return nil
	}

	status := string(sub.Status)
	periodEnd := time.Unix(sub.CurrentPeriodEnd, 0)

	var user users.User
	userID := userIDFromMetadata(sub.Metadata)
	if userID != 0 {
		_ = database.DB.Where("id = ?", userID).First(&user).Error
	}
	if user.ID == 0 {
		_ = database.DB.Where("subscription_id = ?", sub.ID).First(&user).Error
	}
	if user.ID == 0 {
		return nil
	}

	updates := map[string]interface{}{
		"stripe_subscription_status": status,
		"subscription_end":           periodEnd,
		"current_period_end":         periodEnd,
	}

	return database.DB.Model(&users.User{}).
		Where("id = ?", user.ID).
		Updates(updates).Error
}
