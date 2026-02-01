package billing

import (
	"net/http"
	"registration-app/database"
	"registration-app/internal/domain/billing"

	"github.com/gin-gonic/gin"
)

func GetPaymentHistory(c *gin.Context) {
	userID := c.GetUint("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var payments []billing.Payment
	if err := database.DB.
		Preload("Plan").
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&payments).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load payments"})
		return
	}

	c.JSON(http.StatusOK, payments)
}
