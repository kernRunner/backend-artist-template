package admin

import (
	"net/http"
	"registration-app/database"
	"registration-app/internal/domain/billing"
	"registration-app/internal/domain/users"
	"time"

	"github.com/gin-gonic/gin"
)

type AdminUser struct {
	ID                uint       `json:"id"`
	Name              string     `json:"name"`
	Lastname          string     `json:"lastname"`
	Tel               string     `json:"tel"`
	Email             string     `json:"email"`
	Role              string     `json:"role"`
	IsVerified        bool       `json:"is_verified"`
	PlanName          *string    `json:"plan_name,omitempty"`
	StripeCustomerID  *string    `json:"stripe_customer_id,omitempty"`
	StripeSubID       *string    `json:"stripe_subscription_id,omitempty"`
	SubscriptionStart *time.Time `json:"subscription_start,omitempty"`
	SubscriptionEnd   *time.Time `json:"subscription_end,omitempty"`
}

type AdminPayment struct {
	ID         uint    `json:"id"`
	Email      string  `json:"email"`
	PlanName   *string `json:"plan_name,omitempty"`
	AmountEUR  float64 `json:"amount_eur"`
	Status     string  `json:"status"`
	InvoiceID  *string `json:"invoice_id,omitempty"`
	ReceiptURL *string `json:"receipt_url,omitempty"`
	CreatedAt  string  `json:"created_at"`
}

type AdminStats struct {
	TotalUsers    int            `json:"total_users"`
	TotalRevenue  float64        `json:"total_revenue"`
	RecentRevenue float64        `json:"recent_revenue"`
	UsersPerPlan  map[string]int `json:"users_per_plan"`
}

func AdminDashboard(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "Welcome to the admin dashboard 👑",
	})
}

func ListAllUsers(c *gin.Context) {
	var users []users.User
	err := database.DB.Preload("Plan").Find(&users).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load users"})
		return
	}

	var adminUsers []AdminUser
	for _, u := range users {
		var planName *string
		if u.Plan != nil {
			planName = &u.Plan.Name
		}

		adminUsers = append(adminUsers, AdminUser{
			ID:                u.ID,
			Name:              u.Name,
			Lastname:          u.Lastname,
			Tel:               u.Tel,
			Email:             u.Email,
			Role:              u.Role,
			IsVerified:        u.IsVerified,
			PlanName:          planName,
			StripeCustomerID:  u.StripeCustomerID,
			StripeSubID:       u.SubscriptionId,
			SubscriptionStart: u.SubscriptionStart,
			SubscriptionEnd:   u.SubscriptionEnd,
		})
	}

	c.JSON(http.StatusOK, adminUsers)
}

func ListAllPayments(c *gin.Context) {
	var payments []billing.Payment
	err := database.DB.Preload("User").Preload("Plan").Order("created_at DESC").Find(&payments).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load payments"})
		return
	}

	var result []AdminPayment
	for _, p := range payments {
		var planName *string
		if p.Plan != nil {
			planName = &p.Plan.Name
		}
		result = append(result, AdminPayment{
			ID:         p.ID,
			Email:      p.User.Email,
			PlanName:   planName,
			AmountEUR:  p.AmountEUR,
			Status:     p.Status,
			InvoiceID:  p.InvoiceID,
			ReceiptURL: p.ReceiptURL,
			CreatedAt:  p.CreatedAt.Format("2006-01-02 15:04"),
		})
	}

	c.JSON(http.StatusOK, result)
}

func GetAdminStats(c *gin.Context) {
	var stats AdminStats

	// Fix for type mismatch
	var totalUsers int64
	var totalRevenue float64
	var recentRevenue float64

	database.DB.Model(&users.User{}).Count(&totalUsers)
	database.DB.Model(&billing.Payment{}).Where("status = ?", "paid").Select("COALESCE(SUM(amount_eur), 0)").Scan(&totalRevenue)

	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
	database.DB.Model(&billing.Payment{}).
		Where("status = ? AND created_at >= ?", "paid", thirtyDaysAgo).
		Select("COALESCE(SUM(amount_eur), 0)").Scan(&recentRevenue)

	stats.TotalUsers = int(totalUsers)
	stats.TotalRevenue = totalRevenue
	stats.RecentRevenue = recentRevenue

	type PlanCount struct {
		Name  *string
		Count int
	}
	var counts []PlanCount

	database.DB.
		Table("users").
		Select("plans.name, COUNT(users.id) as count").
		Joins("LEFT JOIN plans ON users.plan_id = plans.id").
		Group("plans.name").
		Scan(&counts)

	stats.UsersPerPlan = map[string]int{}
	for _, c := range counts {
		name := "No Plan"
		if c.Name != nil {
			name = *c.Name
		}
		stats.UsersPerPlan[name] = c.Count
	}

	c.JSON(http.StatusOK, stats)
}

func GetUserDetails(c *gin.Context) {
	userID := c.Param("id")

	var user users.User
	if err := database.DB.Preload("Plan").First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	var payments []billing.Payment
	if err := database.DB.Preload("Plan").Where("user_id = ?", userID).Find(&payments).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch payments"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user":     user,
		"payments": payments,
	})
}
