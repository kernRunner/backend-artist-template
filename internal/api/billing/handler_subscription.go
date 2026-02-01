package billing

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"os"
	"strconv"

	"github.com/stripe/stripe-go/v75"
	"github.com/stripe/stripe-go/v75/price"
)

type StripePlan struct {
	PriceID        string  `json:"price_id"`
	ProductID      string  `json:"product_id"`
	Name           string  `json:"name"`
	Currency       string  `json:"currency"`
	UnitAmount     float64 `json:"unit_amount"` // in major units (EUR)
	Interval       string  `json:"interval"`    // month/year
	StorageGB      int     `json:"storage_gb"`
	DurationMonths int     `json:"duration_months"`
}

func ListPlansFromStripe(c *gin.Context) {
	stripe.Key = os.Getenv("STRIPE_SECRET_KEY")
	if stripe.Key == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Stripe key not configured"})
		return
	}

	params := &stripe.PriceListParams{}
	params.Active = stripe.Bool(true)
	params.Type = stripe.String("recurring")
	params.AddExpand("data.product")

	it := price.List(params)

	plans := []StripePlan{}
	for it.Next() {
		p := it.Price()

		// Safety checks
		if !p.Active || p.Recurring == nil {
			continue
		}

		// Product must exist and be active
		if p.Product == nil || !p.Product.Active {
			continue
		}

		// Optional: hide prices via metadata
		if p.Metadata["visible"] == "false" {
			continue
		}

		storageGB, _ := strconv.Atoi(p.Metadata["storage_gb"])
		durationMonths, _ := strconv.Atoi(p.Metadata["duration_months"])

		amount := float64(p.UnitAmount) / 100.0

		plans = append(plans, StripePlan{
			PriceID:        p.ID,
			ProductID:      p.Product.ID,
			Name:           p.Product.Name,
			Currency:       string(p.Currency),
			UnitAmount:     amount,
			Interval:       string(p.Recurring.Interval),
			StorageGB:      storageGB,
			DurationMonths: durationMonths,
		})
	}

	if err := it.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch Stripe prices"})
		return
	}

	c.JSON(http.StatusOK, plans)
}

// func SubscribeToPlan(c *gin.Context) {
// 	email := c.GetString("email")

// 	var body struct {
// 		PlanID int `json:"plan_id"`
// 	}
// 	if err := c.ShouldBindJSON(&body); err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
// 		return
// 	}

// 	var user models.User
// 	if err := database.DB.Where("email = ?", email).First(&user).Error; err != nil {
// 		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
// 		return
// 	}

// 	var plan models.Plan
// 	if err := database.DB.First(&plan, body.PlanID).Error; err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "Plan not found"})
// 		return
// 	}

// 	start := time.Now()
// 	end := start.AddDate(0, plan.DurationMonths, 0)

// 	user.PlanID = &plan.ID
// 	user.SubscriptionStart = &start
// 	user.SubscriptionEnd = &end

// 	if err := database.DB.Save(&user).Error; err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to subscribe to plan"})
// 		return
// 	}

// 	c.JSON(http.StatusOK, gin.H{"message": "Subscription activated", "plan": plan})
// }
