package works

import (
	"registration-app/internal/domain/works"

	"gorm.io/gorm"
)

func userSeriesQuery(db *gorm.DB, userID uint) *gorm.DB {
	return db.Model(&works.Series{}).
		Where("owner_type = ? AND user_id = ?", works.OwnerUser, userID)
}

func userArtworksQuery(db *gorm.DB, userID uint) *gorm.DB {
	return db.Model(&works.Artwork{}).
		Where("owner_type = ? AND user_id = ?", works.OwnerUser, userID)
}

func templateSeriesQuery(db *gorm.DB) *gorm.DB {
	// if user_id is nullable in DB, you can also add: AND user_id IS NULL
	return db.Model(&works.Series{}).
		Where("owner_type = ?", works.OwnerSystem)
}

func templateArtworksQuery(db *gorm.DB) *gorm.DB {
	return db.Model(&works.Artwork{}).
		Where("owner_type = ?", works.OwnerSystem)
}
