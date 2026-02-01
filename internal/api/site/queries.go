package siteapi

import (
	"registration-app/internal/domain/site"

	"gorm.io/gorm"
)

func userPagesQuery(db *gorm.DB, userID uint) *gorm.DB {
	return db.Model(&site.SitePage{}).
		Where("owner_type = ? AND user_id = ?", site.OwnerUser, userID)
}

func templatePagesQuery(db *gorm.DB, templateID string) *gorm.DB {
	return db.Model(&site.SitePage{}).
		Where("owner_type = ? AND template_id = ?", site.OwnerSystem, templateID)
}
