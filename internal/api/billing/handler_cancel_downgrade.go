package billing

import (
	"net/http"
	"os"

	"registration-app/database"
	"registration-app/internal/domain/users"

	"github.com/gin-gonic/gin"
	"github.com/stripe/stripe-go/v75"
	schedules "github.com/stripe/stripe-go/v75/subscriptionschedule"
)

func CancelDowngrade(c *gin.Context) {
	stripe.Key = os.Getenv("STRIPE_SECRET_KEY")
	if stripe.Key == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Stripe key not configured"})
		return
	}

	email := c.GetString("email")
	if email == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not identified"})
		return
	}

	var user users.User
	if err := database.DB.Where("email = ?", email).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	// Nothing scheduled? Then nothing to cancel.
	if user.StripeScheduleID == nil || *user.StripeScheduleID == "" || user.PendingPlanID == nil {
		c.JSON(http.StatusOK, gin.H{"message": "No pending downgrade to cancel"})
		return
	}

	scheduleID := *user.StripeScheduleID

	// ✅ Release schedule so the subscription continues normally on the current plan
	_, err := schedules.Release(scheduleID, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to release Stripe schedule",
			"details": err.Error(),
		})
		return
	}

	// ✅ Clear pending downgrade fields locally
	if err := database.DB.Model(&users.User{}).
		Where("id = ?", user.ID).
		Updates(map[string]interface{}{
			"pending_plan_id":         nil,
			"pending_plan_start_date": nil,
			"stripe_schedule_id":      nil,
		}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to clear pending downgrade in DB",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "Pending downgrade cancelled",
		"schedule_id": scheduleID,
	})
}
