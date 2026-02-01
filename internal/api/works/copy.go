package works

import (
	"net/http"

	"registration-app/database"
	"registration-app/internal/domain/media"
	dw "registration-app/internal/domain/works"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// POST /templates/series/:id/copy
func CopyTemplateSeriesToUser(c *gin.Context) {
	templateSeriesID := c.Param("id")

	userID, ok := mustUserID(c)
	if !ok {
		return
	}
	uid := userID

	var newSeriesID string

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		// 1) Load template series identity + published revision + items + item published revisions
		var tpl dw.Series
		if err := tx.
			Preload("PublishedRevision.Image").
			Preload("PublishedRevision.I18n").
			Preload("Items", func(db *gorm.DB) *gorm.DB {
				return db.Order("sort_index ASC")
			}).
			Preload("Items.PublishedRevision.Image").
			Preload("Items.PublishedRevision.I18n").
			First(&tpl, "id = ? AND owner_type = ?", templateSeriesID, dw.OwnerSystem).Error; err != nil {
			return err
		}

		if tpl.PublishedRevision == nil {
			return gorm.ErrRecordNotFound // or custom error: "template has no published revision"
		}

		// 2) Create user-owned series identity
		newSeries := dw.Series{
			OwnerType: dw.OwnerUser,
			UserID:    &uid,
			IDLocked:  false,
		}
		if err := tx.Create(&newSeries).Error; err != nil {
			return err
		}
		newSeriesID = newSeries.ID

		// 3) Copy series revision image (duplicate image row)
		var newSeriesImageID *string
		if tpl.PublishedRevision.Image != nil {
			img := media.Image{
				OriginalPath: tpl.PublishedRevision.Image.OriginalPath,
				WebpPath:     tpl.PublishedRevision.Image.WebpPath,
				AvifPath:     tpl.PublishedRevision.Image.AvifPath,
			}
			if err := tx.Create(&img).Error; err != nil {
				return err
			}
			newSeriesImageID = &img.ID
		}

		// 4) Create draft revision for the new series
		newSeriesRev := dw.SeriesRevision{
			SeriesID: newSeries.ID,
			ImageID:  newSeriesImageID,
		}
		if err := tx.Create(&newSeriesRev).Error; err != nil {
			return err
		}

		for _, row := range tpl.PublishedRevision.I18n {
			out := dw.SeriesI18nRevision{
				SeriesRevisionID: newSeriesRev.ID,
				Lang:             row.Lang,
				Title:            row.Title,
				DescriptionSerie: row.DescriptionSerie,
				Year:             row.Year,
			}
			if err := tx.Create(&out).Error; err != nil {
				return err
			}
		}

		// point series.draft_revision_id to this new revision
		if err := tx.Model(&dw.Series{}).
			Where("id = ?", newSeries.ID).
			Update("draft_revision_id", newSeriesRev.ID).Error; err != nil {
			return err
		}

		// 5) Copy artworks identities + draft revisions
		for _, art := range tpl.Items {
			// create artwork identity
			newArt := dw.Artwork{
				OwnerType: dw.OwnerUser,
				UserID:    &uid,
				SeriesID:  newSeries.ID,
				SortIndex: art.SortIndex,
				IDLocked:  false,
				Sold:      false,
			}
			if err := tx.Create(&newArt).Error; err != nil {
				return err
			}

			// copy artwork revision image
			var newArtImageID *string
			if art.PublishedRevision != nil && art.PublishedRevision.Image != nil {
				img := media.Image{
					OriginalPath: art.PublishedRevision.Image.OriginalPath,
					WebpPath:     art.PublishedRevision.Image.WebpPath,
					AvifPath:     art.PublishedRevision.Image.AvifPath,
				}
				if err := tx.Create(&img).Error; err != nil {
					return err
				}
				newArtImageID = &img.ID
			}

			// create draft revision
			newArtRev := dw.ArtworkRevision{
				ArtworkID: newArt.ID,
				ImageID:   newArtImageID,
			}
			if art.PublishedRevision != nil {
				newArtRev.Year = art.PublishedRevision.Year
				newArtRev.Medium = art.PublishedRevision.Medium
				newArtRev.SizeCM = art.PublishedRevision.SizeCM
				newArtRev.Price = art.PublishedRevision.Price
			}

			if err := tx.Create(&newArtRev).Error; err != nil {
				return err
			}

			// i18n copy
			if art.PublishedRevision != nil {
				for _, t := range art.PublishedRevision.I18n {
					out := dw.ArtworkI18nRevision{
						ArtworkRevisionID: newArtRev.ID,
						Lang:              t.Lang,
						Title:             t.Title,
						Description:       t.Description,
						Notes:             t.Notes,
					}
					if err := tx.Create(&out).Error; err != nil {
						return err
					}
				}
			}

			// point artwork.draft_revision_id
			if err := tx.Model(&dw.Artwork{}).
				Where("id = ?", newArt.ID).
				Update("draft_revision_id", newArtRev.ID).Error; err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Template series not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to copy template", "details": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"series_id": newSeriesID,
	})
}
