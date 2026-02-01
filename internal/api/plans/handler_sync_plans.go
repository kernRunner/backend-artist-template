package plans

import (
	"net/http"
	"os"

	"registration-app/database"
	"registration-app/internal/domain/plans"

	"github.com/gin-gonic/gin"
	"github.com/stripe/stripe-go/v75"
	"github.com/stripe/stripe-go/v75/price"
)

func SyncPlansFromStripe(c *gin.Context) {
	stripe.Key = os.Getenv("STRIPE_SECRET_KEY")
	if stripe.Key == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Stripe key not configured"})
		return
	}

	targetProductID := os.Getenv("STRIPE_PORTFOLIO_PRODUCT_ID") // recommended
	// targetProductName := "Artist Portfolio" // fallback if you don't want env var

	params := &stripe.PriceListParams{}
	params.Active = stripe.Bool(true)
	params.Type = stripe.String("recurring")
	params.AddExpand("data.product")

	it := price.List(params)

	synced := 0
	created := 0
	updated := 0
	skipped := 0

	for it.Next() {
		p := it.Price()

		if !p.Active || p.Recurring == nil || p.Product == nil || !p.Product.Active {
			skipped++
			continue
		}

		// ✅ filter by product (recommended)
		if targetProductID != "" && p.Product.ID != targetProductID {
			skipped++
			continue
		}
		// OR filter by name (less stable)
		// if p.Product.Name != targetProductName { skipped++; continue }

		// ✅ optionally keep only EUR
		if string(p.Currency) != "eur" {
			skipped++
			continue
		}

		// visibility flag
		if p.Metadata != nil && p.Metadata["visible"] == "false" {
			skipped++
			continue
		}

		amount := float64(p.UnitAmount) / 100.0

		// ✅ use metadata for display name / key / tier
		displayName := p.Product.Name
		if p.Metadata != nil {
			if v := p.Metadata["plan"]; v != "" {
				displayName = v
			}
		}

		tier := "" // store it to DB if you want
		if p.Metadata != nil {
			if v := p.Metadata["plan"]; v != "" { // your requested key
				tier = v // "essential|professional|advanced"
			} else if v := p.Metadata["tier"]; v != "" {
				tier = v
			}
		}

		// Try find existing plan by stripe price id
		var existing plans.Plan
		err := database.DB.Where("stripe_price_id = ?", p.ID).First(&existing).Error

		if err != nil {
			plan := plans.Plan{
				Name:          displayName,
				PriceEUR:      amount,
				StripePriceID: p.ID,
				Interval:      string(p.Recurring.Interval),
				Tier:          tier,
			}
			if err := database.DB.Create(&plan).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create plan", "details": err.Error()})
				return
			}
			created++
		} else {
			existing.Name = displayName
			existing.PriceEUR = amount
			existing.Interval = string(p.Recurring.Interval)
			if tier != "" {
				existing.Tier = tier
			}

			if err := database.DB.Save(&existing).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update plan", "details": err.Error()})
				return
			}
			updated++
		}

		synced++
	}

	if err := it.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch Stripe prices", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"synced":  synced,
		"created": created,
		"updated": updated,
		"skipped": skipped,
	})
}

func ListPlans(c *gin.Context) {
	targetProductID := os.Getenv("STRIPE_PORTFOLIO_PRODUCT_ID")

	var plansList []plans.Plan
	q := database.DB.Model(&plans.Plan{})

	if targetProductID != "" {
		q = q.Where("stripe_product_id = ?", targetProductID)
	}

	q = q.Order("price_eur ASC")

	if err := q.Find(&plansList).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load plans"})
		return
	}

	c.JSON(http.StatusOK, plansList)
}
