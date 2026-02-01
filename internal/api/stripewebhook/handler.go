package stripewebhooks

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"registration-app/database"

	"github.com/gin-gonic/gin"
	"github.com/stripe/stripe-go/v75"
	"github.com/stripe/stripe-go/v75/webhook"
)

func StripeWebhook(c *gin.Context) {
	// Stripe key is required for any follow-up API calls (checkoutsession.Get, subscription.Get, etc.)
	stripe.Key = os.Getenv("STRIPE_SECRET_KEY")
	if stripe.Key == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "STRIPE_SECRET_KEY not configured"})
		return
	}

	endpointSecret := os.Getenv("STRIPE_WEBHOOK_SECRET")
	if endpointSecret == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "STRIPE_WEBHOOK_SECRET not configured"})
		return
	}

	payload, err := readStripeBody(c, 65536)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Error reading request body"})
		return
	}

	event, err := webhook.ConstructEventWithOptions(
		payload,
		c.GetHeader("Stripe-Signature"),
		endpointSecret,
		webhook.ConstructEventOptions{IgnoreAPIVersionMismatch: true},
	)
	if err != nil {
		fmt.Println("❌ Stripe signature verification failed:", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Signature verification failed"})
		return
	}

	// Optional: store event.ID for idempotency (recommended)
	// If you add a stripe_events table, check if event.ID already processed and return 200.

	switch event.Type {
	case "checkout.session.completed":
		var session stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse session"})
			return
		}
		if err := handleCheckoutSessionCompleted(c, &session); err != nil {
			// Return 200 for non-retryable errors; 500 for retryable.
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "received"})
		return

	case "customer.subscription.updated":
		var sub stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse subscription"})
			return
		}
		if err := handleSubscriptionUpdated(c, &sub); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "received"})
		return

	case "customer.subscription.deleted":
		var sub stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse subscription"})
			return
		}
		if err := handleSubscriptionDeleted(c, &sub); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "received"})
		return

	default:
		// Acknowledge unknown events to avoid retries
		c.JSON(http.StatusOK, gin.H{"status": "ignored"})
		return
	}

	_ = database.DB // silence if unused in your editor
}

func readStripeBody(c *gin.Context, maxBytes int64) ([]byte, error) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
	return io.ReadAll(c.Request.Body)
}
