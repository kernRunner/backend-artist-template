package middleware

import (
	"net/http"
	"registration-app/database"
	"registration-app/internal/domain/users"
	"time"

	"github.com/gin-gonic/gin"
)

func RequireActiveSubscription() gin.HandlerFunc {
	return func(c *gin.Context) {
		email := c.GetString("email")
		var user users.User

		// GORM query by email
		if err := database.DB.Where("email = ?", email).First(&user).Error; err != nil || user.SubscriptionEnd == nil {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "Subscription not found or expired",
			})
			return
		}

		// Check expiration
		if time.Now().After(*user.SubscriptionEnd) {
			c.AbortWithStatusJSON(http.StatusPaymentRequired, gin.H{
				"error": "Your subscription has expired",
			})
			return
		}

		c.Next()
	}
}
