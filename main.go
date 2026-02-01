package main

import (
	"os"
	"registration-app/config"
	"registration-app/database"
	routes "registration-app/internal/app/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	// gin.SetMode(gin.ReleaseMode) uncomment only in production
	config.LoadEnv()
	database.InitDB()

	r := gin.Default()

	// ✅ Add CORS middleware BEFORE registering routes
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{os.Getenv("CORS_ORIGIN")},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	routes.RegisterRoutes(r)

	r.Run(":" + config.PORT)
}
