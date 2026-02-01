package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

var (
	PORT       string
	DB_URL     string
	JWT_SECRET string

	GOOGLE_CLIENT_ID         string
	GOOGLE_CLIENT_SECRET     string
	GOOGLE_REDIRECT_URL      string
	GOOGLE_FRONTEND_REDIRECT string
)

func LoadEnv() {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found. Using system environment variables.")
	}

	PORT = getEnv("PORT", "8080")
	DB_URL = mustEnv("DB_URL")
	JWT_SECRET = mustEnv("JWT_SECRET")

	// ✅ ADD THESE
	GOOGLE_CLIENT_ID = mustEnv("GOOGLE_CLIENT_ID")
	GOOGLE_CLIENT_SECRET = mustEnv("GOOGLE_CLIENT_SECRET")
	GOOGLE_REDIRECT_URL = mustEnv("GOOGLE_REDIRECT_URL")
	GOOGLE_FRONTEND_REDIRECT = getEnv("GOOGLE_FRONTEND_REDIRECT", "")
}

func mustEnv(key string) string {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		log.Fatalf("Missing required environment variable: %s", key)
	}
	return v
}

func getEnv(key string, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
