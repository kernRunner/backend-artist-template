package works

import (
	"registration-app/internal/domain/media"
	"registration-app/internal/domain/works"

	"gorm.io/gorm"
)

func ensureDraftSeriesRevision(tx *gorm.DB, s *works.Series) (*works.SeriesRevision, error) {
	// draft already exists
	if s.DraftRevisionID != nil && *s.DraftRevisionID != "" {
		var dr works.SeriesRevision
		if err := tx.Preload("I18n").First(&dr, "id = ?", *s.DraftRevisionID).Error; err != nil {
			return nil, err
		}
		return &dr, nil
	}

	// clone from published if available
	var base *works.SeriesRevision
	if s.PublishedRevisionID != nil && *s.PublishedRevisionID != "" {
		var pr works.SeriesRevision
		if err := tx.Preload("I18n").First(&pr, "id = ?", *s.PublishedRevisionID).Error; err != nil {
			return nil, err
		}
		base = &pr
	}

	dr := works.SeriesRevision{SeriesID: s.ID}
	if base != nil {
		dr.ImageID = base.ImageID
	}
	if err := tx.Create(&dr).Error; err != nil {
		return nil, err
	}

	// copy i18n
	if base != nil {
		for _, t := range base.I18n {
			row := works.SeriesI18nRevision{
				SeriesRevisionID: dr.ID,
				Lang:             t.Lang,
				Title:            t.Title,
				DescriptionSerie: t.DescriptionSerie,
				Year:             t.Year,
			}
			if err := tx.Create(&row).Error; err != nil {
				return nil, err
			}
		}
	}

	if err := tx.Model(&works.Series{}).
		Where("id = ?", s.ID).
		Update("draft_revision_id", dr.ID).Error; err != nil {
		return nil, err
	}

	return &dr, nil
}

func ensureDraftArtworkRevision(tx *gorm.DB, a *works.Artwork) (*works.ArtworkRevision, error) {
	// 1) if we already have a real draft, use it
	if a.DraftRevisionID != nil && *a.DraftRevisionID != "" {
		if a.PublishedRevisionID == nil || *a.PublishedRevisionID == "" || *a.DraftRevisionID != *a.PublishedRevisionID {
			var dr works.ArtworkRevision
			if err := tx.Preload("I18n").First(&dr, "id = ?", *a.DraftRevisionID).Error; err != nil {
				return nil, err
			}
			return &dr, nil
		}
	}

	// 2) otherwise clone published if exists
	if a.PublishedRevisionID != nil && *a.PublishedRevisionID != "" {
		// load published revision (+i18n)
		var pr works.ArtworkRevision
		if err := tx.Preload("I18n").First(&pr, "id = ?", *a.PublishedRevisionID).Error; err != nil {
			return nil, err
		}

		// create new draft revision copying fields
		dr := works.ArtworkRevision{
			ArtworkID: a.ID,
			Year:      pr.Year,
			Medium:    pr.Medium,
			SizeCM:    pr.SizeCM,
			Price:     pr.Price,
			ImageID:   pr.ImageID,
		}
		if err := tx.Create(&dr).Error; err != nil {
			return nil, err
		}

		// clone i18n rows
		for _, t := range pr.I18n {
			row := works.ArtworkI18nRevision{
				ArtworkRevisionID: dr.ID,
				Lang:              t.Lang,
				Title:             t.Title,
				Description:       t.Description,
				Notes:             t.Notes,
			}
			if err := tx.Create(&row).Error; err != nil {
				return nil, err
			}
		}

		// IMPORTANT: point artwork to this new draft
		if err := tx.Model(&works.Artwork{}).
			Where("id = ?", a.ID).
			Update("draft_revision_id", dr.ID).Error; err != nil {
			return nil, err
		}

		// update the struct in memory too (optional but avoids confusion)
		a.DraftRevisionID = &dr.ID

		return &dr, nil
	}

	// 3) no published: create empty draft
	dr := works.ArtworkRevision{ArtworkID: a.ID}
	if err := tx.Create(&dr).Error; err != nil {
		return nil, err
	}
	if err := tx.Model(&works.Artwork{}).Where("id = ?", a.ID).Update("draft_revision_id", dr.ID).Error; err != nil {
		return nil, err
	}
	a.DraftRevisionID = &dr.ID
	return &dr, nil
}

// small helper for image upsert pattern
func upsertImage(tx *gorm.DB, currentImageID *string, inOriginal string, inWebp *string, inAvif *string) (*string, error) {
	if currentImageID != nil && *currentImageID != "" {
		if err := tx.Model(&media.Image{}).
			Where("id = ?", *currentImageID).
			Updates(map[string]interface{}{
				"original_path": inOriginal,
				"webp_path":     inWebp,
				"avif_path":     inAvif,
			}).Error; err != nil {
			return nil, err
		}
		return currentImageID, nil
	}

	img := media.Image{
		OriginalPath: inOriginal,
		WebpPath:     inWebp,
		AvifPath:     inAvif,
	}
	if err := tx.Create(&img).Error; err != nil {
		return nil, err
	}
	return &img.ID, nil
}
