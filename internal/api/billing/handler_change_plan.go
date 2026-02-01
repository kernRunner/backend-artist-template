package billing

import (
	"net/http"
	"os"
	"time"

	"registration-app/database"
	"registration-app/internal/domain/plans"
	"registration-app/internal/domain/users"

	"github.com/gin-gonic/gin"
	"github.com/stripe/stripe-go/v75"
	stripesub "github.com/stripe/stripe-go/v75/subscription"
	schedules "github.com/stripe/stripe-go/v75/subscriptionschedule"
)

func ChangePlan(c *gin.Context) {
	var body struct {
		PriceID string `json:"price_id"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.PriceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing or invalid price_id"})
		return
	}

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

	// Load user + current plan
	var user users.User
	if err := database.DB.Preload("Plan").Where("email = ?", email).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	// Load target plan by stripe price id
	var targetPlan plans.Plan
	if err := database.DB.Where("stripe_price_id = ?", body.PriceID).First(&targetPlan).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Target plan not found in DB (run /admin/sync-plans)"})
		return
	}

	if user.SubscriptionId == nil || *user.SubscriptionId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No active subscription to change. Use checkout first."})
		return
	}

	// Fetch subscription
	sub, err := stripesub.Get(*user.SubscriptionId, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch Stripe subscription", "details": err.Error()})
		return
	}
	if sub.Items == nil || len(sub.Items.Data) == 0 || sub.Items.Data[0].Price == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Subscription has no price item"})
		return
	}

	item := sub.Items.Data[0]
	currentPriceID := item.Price.ID
	if currentPriceID == targetPlan.StripePriceID {
		c.JSON(http.StatusOK, gin.H{"message": "Already on this plan"})
		return
	}

	// Determine upgrade vs downgrade (based on your DB prices)
	isUpgrade := true
	if user.Plan != nil {
		isUpgrade = targetPlan.PriceEUR > user.Plan.PriceEUR
	}

	// -------------------------
	// ✅ UPGRADE: effective now
	// -------------------------
	if isUpgrade {
		updateParams := &stripe.SubscriptionParams{
			Items: []*stripe.SubscriptionItemsParams{
				{
					ID:    stripe.String(item.ID),
					Price: stripe.String(targetPlan.StripePriceID),
				},
			},
			ProrationBehavior: stripe.String("create_prorations"),
		}

		updatedSub, err := stripesub.Update(*user.SubscriptionId, updateParams)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to upgrade subscription", "details": err.Error()})
			return
		}

		periodEnd := time.Unix(updatedSub.CurrentPeriodEnd, 0)
		now := time.Now()

		if err := database.DB.Model(&users.User{}).
			Where("id = ?", user.ID).
			Updates(map[string]interface{}{
				"plan_id":                 targetPlan.ID,
				"subscription_start":      now,
				"subscription_end":        periodEnd,
				"current_period_end":      periodEnd,
				"pending_plan_id":         nil,
				"pending_plan_start_date": nil,
			}).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user in DB", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message":            "Upgraded now (prorated automatically by Stripe)",
			"is_upgrade":         true,
			"current_period_end": periodEnd,
			"subscription_id":    updatedSub.ID,
		})
		return
	}

	// -----------------------------------
	// ✅ DOWNGRADE: schedule next cycle
	// -----------------------------------
	// -----------------------------------
	// ✅ DOWNGRADE: schedule next cycle
	// -----------------------------------
	periodStartUnix := sub.CurrentPeriodStart
	periodEndUnix := sub.CurrentPeriodEnd
	effectiveAt := time.Unix(periodEndUnix, 0)

	scheduleID := ""
	if sub.Schedule != nil {
		scheduleID = sub.Schedule.ID
	}

	if scheduleID == "" {
		// create schedule only if none exists
		schedule, err := schedules.New(&stripe.SubscriptionScheduleParams{
			FromSubscription: stripe.String(sub.ID),
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create schedule", "details": err.Error()})
			return
		}
		scheduleID = schedule.ID
	}

	// now UPDATE schedule phases
	_, err = schedules.Update(scheduleID, &stripe.SubscriptionScheduleParams{
		EndBehavior: stripe.String("release"),
		Phases: []*stripe.SubscriptionSchedulePhaseParams{
			{
				StartDate: stripe.Int64(periodStartUnix),
				EndDate:   stripe.Int64(periodEndUnix),
				Items: []*stripe.SubscriptionSchedulePhaseItemParams{
					{Price: stripe.String(currentPriceID), Quantity: stripe.Int64(1)},
				},
			},
			{
				StartDate: stripe.Int64(periodEndUnix),
				Items: []*stripe.SubscriptionSchedulePhaseItemParams{
					{Price: stripe.String(targetPlan.StripePriceID), Quantity: stripe.Int64(1)},
				},
			},
		},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update schedule phases", "details": err.Error()})
		return
	}

	// 3) Store pending downgrade in DB (keep current plan_id until effectiveAt)
	if err := database.DB.Model(&users.User{}).
		Where("id = ?", user.ID).
		Updates(map[string]interface{}{
			"pending_plan_id":         targetPlan.ID,
			"pending_plan_start_date": effectiveAt,
			"stripe_schedule_id":      scheduleID,
			"current_period_end":      effectiveAt,
			"subscription_end":        effectiveAt,
		}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to store pending downgrade",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":      "Downgrade scheduled for next billing cycle",
		"is_upgrade":   false,
		"effective_at": effectiveAt,
		"schedule_id":  scheduleID,
	})
	return

}
