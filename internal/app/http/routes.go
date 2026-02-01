package routes

import (
	adminapi "registration-app/internal/api/admin"
	authapi "registration-app/internal/api/auth"
	"registration-app/internal/api/billing"
	"registration-app/internal/api/plans"
	siteapi "registration-app/internal/api/site"
	stripewebhooks "registration-app/internal/api/stripewebhook"
	"registration-app/internal/api/users"
	worksapi "registration-app/internal/api/works"
	"registration-app/internal/app/http/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine) {
	// ✅ Apply input sanitization to public routes only

	r.POST("/webhook", stripewebhooks.StripeWebhook)
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	r.GET("/templates/site", siteapi.ListSiteTemplates)
	r.GET("/templates/site/:slug", siteapi.GetSiteTemplate)

	public := r.Group("/")
	public.Use(middleware.SanitizeAndCleanInputMiddleware())

	public.POST("/register", authapi.Register)
	public.POST("/login", authapi.Login)
	public.GET("/plans", plans.ListPlans)
	public.GET("/verify", users.VerifyEmail)
	public.POST("/resend-verification", authapi.ResendVerification)
	public.POST("/request-password-reset", authapi.RequestPasswordReset)
	public.POST("/reset-password", authapi.ResetPassword)

	public.GET("/auth/google", authapi.GoogleStart)
	public.GET("/auth/google/callback", authapi.GoogleCallback)

	// Authenticated
	auth := r.Group("/")
	auth.Use(middleware.AuthMiddleware())
	auth.GET("/me", users.GetCurrentUser)
	auth.GET("/payments", billing.GetPaymentHistory)
	auth.POST("/create-checkout-session", billing.CreateCheckoutSession)
	auth.POST("/billing-portal", billing.CreateBillingPortal)
	auth.POST("/change-password", authapi.ChangePassword)
	auth.POST("/cancel-downgrade", billing.CancelDowngrade)

	auth.GET("/works", worksapi.GetWorksJSON)
	auth.GET("/templates/works", worksapi.GetTemplateWorksJSON)

	auth.GET("/series/:id", worksapi.GetSeriesByID)
	auth.GET("/artworks/:id", worksapi.GetArtworkByID)

	auth.POST("/series", worksapi.CreateSeries)
	auth.PUT("/series/:id", worksapi.UpdateSeries)
	auth.DELETE("/series/:id", worksapi.DeleteSeries)

	auth.POST("/series/:id/publish", worksapi.PublishSeries)
	auth.POST("/series/:id/unpublish", worksapi.UnpublishSeries)

	auth.POST("/series/:id/artworks", worksapi.CreateArtwork)
	auth.DELETE("/series/:id/artworks", worksapi.DeleteAllArtworksOfSeries)
	auth.PUT("/artworks/:id", worksapi.UpdateArtwork)
	auth.DELETE("/artworks/:id", worksapi.DeleteArtwork)

	auth.POST("/artworks/:id/publish", worksapi.PublishArtwork)
	auth.POST("/artworks/:id/unpublish", worksapi.UnpublishArtwork)

	auth.PUT("/series/:id/artworks/reorder", worksapi.ReorderArtworks)

	auth.POST("/templates/series/:id/copy", worksapi.CopyTemplateSeriesToUser)

	auth.GET("/site", siteapi.GetUserSite)
	auth.POST("/site/from-template/:slug", siteapi.CopySiteFromTemplate)

	auth.POST("/series/:id/artworks/discard-drafts", worksapi.BulkDiscardArtworkDrafts)

	// Subscribed users
	subscribed := auth.Group("/")
	subscribed.Use(middleware.RequireActiveSubscription())
	subscribed.POST("/change-plan", billing.ChangePlan)

	// Admin routes
	admin := r.Group("/admin")
	admin.Use(middleware.AuthMiddleware(), middleware.RequireRole("admin"))
	admin.GET("/dashboard", adminapi.AdminDashboard)
	admin.GET("/users", adminapi.ListAllUsers)
	admin.GET("/payments", adminapi.ListAllPayments)
	admin.GET("/user/:id", adminapi.GetUserDetails)
	admin.POST("/sync-plans", plans.SyncPlansFromStripe)
	admin.POST("/templates/series", worksapi.CreateTemplateSeries)
	admin.POST("/templates/series/:id/artworks", worksapi.CreateTemplateArtwork)

}
