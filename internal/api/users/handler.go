package users

import (
	"net/http"
	"registration-app/database"
	"registration-app/internal/domain/access"
	"registration-app/internal/domain/site"
	"registration-app/internal/domain/users"
	"time"

	"github.com/gin-gonic/gin"
)

func GetCurrentUser(c *gin.Context) {
	email := c.GetString("email")
	if email == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var user users.User
	if err := database.DB.
		Preload("Plan").
		Preload("PendingPlan").
		Where("email = ?", email).
		First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	now := time.Now()
	_, _ = site.EnsureSiteSlug(database.DB, &user)

	policy := access.ComputePolicy(now, user)

	// map limits -> dto (only when not nil)
	var limits *LimitsDTO
	if policy.Limits != nil {
		limits = &LimitsDTO{
			MaxArtworks:          policy.Limits.MaxArtworks,
			HideGalleries:        policy.Limits.HideGalleries,
			NoIndex:              policy.Limits.NoIndex,
			ShowPlatformBranding: policy.Limits.ShowPlatformBranding,
		}
	}

	resp := MeResponse{
		User: UserDTO{
			ID:         user.ID,
			Email:      user.Email,
			Name:       user.Name,
			Lastname:   user.Lastname,
			Tel:        stringPtrIfNotEmpty(user.Tel),
			Role:       user.Role,
			IsVerified: user.IsVerified,
		},
		Billing: BillingDTO{
			Plan:          BuildPlanDTO(user.Plan),
			Subscription:  BuildSubscriptionDTO(user),
			Trial:         BuildTrialDTO(now, user.TrialStartAt, user.TrialEndAt),
			PendingChange: BuildPendingChangeDTO(user),
		},
		Access: AccessDTO{
			State:        string(policy.State),
			Capabilities: policy.Capabilities,
			Site:         BuildAccessSiteDTO(user, policy, limits),
		},
	}

	c.JSON(http.StatusOK, resp)
}

func stringPtrIfNotEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func VerifyEmail(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing token"})
		return
	}

	type Token struct {
		UserID int
	}
	var t Token
	if err := database.DB.Table("verification_tokens").Where("token = ?", token).First(&t).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or expired token"})
		return
	}

	if err := database.DB.Model(&users.User{}).Where("id = ?", t.UserID).Update("is_verified", true).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify user"})
		return
	}

	_ = database.DB.Exec("DELETE FROM verification_tokens WHERE token = ?", token)

	redirectURL := "http://localhost:5173/signin"
	c.Redirect(http.StatusTemporaryRedirect, redirectURL)
}
