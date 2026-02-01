package billing

import (
	"fmt"
	"net/http"
	"os"
	"registration-app/database"
	"registration-app/internal/domain/plans"
	"registration-app/internal/domain/users"

	"github.com/gin-gonic/gin"
	"github.com/stripe/stripe-go/v75"
	portalSession "github.com/stripe/stripe-go/v75/billingportal/session"
	checkoutsession "github.com/stripe/stripe-go/v75/checkout/session"
	customer "github.com/stripe/stripe-go/v75/customer"
)

func CreateCheckoutSession(c *gin.Context) {
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

	userID := c.GetUint("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not identified"})
		return
	}

	// allow-list price id
	var plan plans.Plan
	if err := database.DB.Where("stripe_price_id = ?", body.PriceID).First(&plan).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unknown plan/price_id"})
		return
	}

	var user users.User
	if err := database.DB.Where("id = ?", userID).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}

	if !user.IsVerified {
		c.JSON(http.StatusForbidden, gin.H{"error": "Please verify your email first"})
		return
	}

	// ensure stripe customer
	if user.StripeCustomerID == nil || *user.StripeCustomerID == "" {
		cus, err := customer.New(&stripe.CustomerParams{
			Email: stripe.String(user.Email),
			Metadata: map[string]string{
				"user_id": fmt.Sprint(user.ID),
				"app_env": os.Getenv("APP_ENV"),
			},
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create Stripe customer"})
			return
		}

		if err := database.DB.Model(&users.User{}).
			Where("id = ?", user.ID).
			Update("stripe_customer_id", cus.ID).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store Stripe customer"})
			return
		}

		user.StripeCustomerID = stripe.String(cus.ID)
	}

	appURL := os.Getenv("APP_URL")
	if appURL == "" {
		appURL = "http://localhost:5173"
	}

	params := &stripe.CheckoutSessionParams{
		SuccessURL: stripe.String(appURL + "/account"),
		CancelURL:  stripe.String(appURL + "/account?canceled=1"),
		Mode:       stripe.String(string(stripe.CheckoutSessionModeSubscription)),
		Customer:   stripe.String(*user.StripeCustomerID),

		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{Price: stripe.String(plan.StripePriceID), Quantity: stripe.Int64(1)},
		},

		ClientReferenceID: stripe.String(fmt.Sprint(user.ID)),

		SubscriptionData: &stripe.CheckoutSessionSubscriptionDataParams{
			Metadata: map[string]string{
				"user_id": fmt.Sprint(user.ID),
				"plan_id": fmt.Sprint(plan.ID),
			},
		},
	}

	s, err := checkoutsession.New(params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create checkout session", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"url": s.URL})
}

func CreateBillingPortal(c *gin.Context) {
	stripe.Key = os.Getenv("STRIPE_SECRET_KEY")
	if stripe.Key == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Stripe key not configured"})
		return
	}

	userID := c.GetUint("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not identified"})
		return
	}

	var user users.User
	if err := database.DB.Where("id = ?", userID).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found"})
		return
	}
	if user.StripeCustomerID == nil || *user.StripeCustomerID == "" {
		c.JSON(http.StatusConflict, gin.H{"error": "No Stripe customer yet (subscribe first)"})
		return
	}

	appURL := os.Getenv("APP_URL")
	if appURL == "" {
		appURL = "http://localhost:5173"
	}

	portal, err := portalSession.New(&stripe.BillingPortalSessionParams{
		Customer:  stripe.String(*user.StripeCustomerID),
		ReturnURL: stripe.String(appURL + "/account"),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create billing portal session", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"url": portal.URL})
}
