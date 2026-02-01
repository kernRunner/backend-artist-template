package database

import (
	"fmt"
	"log"
	"os"

	"registration-app/internal/domain/billing"
	"registration-app/internal/domain/media"
	"registration-app/internal/domain/plans"
	"registration-app/internal/domain/site"
	"registration-app/internal/domain/users"
	"registration-app/internal/domain/works"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func InitDB() {
	dsn := os.Getenv("DB_URL")
	if dsn == "" {
		log.Fatal("❌ DB_URL not set")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("❌ Failed to connect to database:", err)
	}

	DB = db

	// ✅ REQUIRED for UUID generation
	if err := DB.Exec(`CREATE EXTENSION IF NOT EXISTS pgcrypto;`).Error; err != nil {
		log.Fatal("❌ Failed to enable pgcrypto extension:", err)
	}

	// ✅ Auto-migrate all domain models
	if err := DB.AutoMigrate(
		// core
		&users.User{},
		&users.VerificationToken{},
		&plans.Plan{},
		&billing.Payment{},

		// media
		&media.Image{},

		// works (NEW)
		&works.Series{},
		&works.SeriesRevision{},
		&works.SeriesI18nRevision{},
		&works.Artwork{},
		&works.ArtworkRevision{},
		&works.ArtworkI18nRevision{},

		// site
		&site.Template{},
		&site.SitePage{},
		&site.SitePageBlock{},
	); err != nil {
		log.Fatal("❌ AutoMigrate error:", err)
	}

	fmt.Println("✅ Connected and migrated successfully")
}
