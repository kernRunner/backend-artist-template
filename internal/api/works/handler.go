package works

import (
	"fmt"
	"net/http"

	"registration-app/database"
	"registration-app/internal/domain/works"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func mustUserID(c *gin.Context) (uint, bool) {
	userID := c.GetUint("user_id")
	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return 0, false
	}
	return userID, true
}

// ------------------------------
// GET /works  -> WorksJSON shape
// ------------------------------
func GetWorksJSON(c *gin.Context) {
	userID, ok := mustUserID(c)
	if !ok {
		return
	}

	view := c.DefaultQuery("view", "draft") // "draft" | "published"

	var series []works.Series
	err := userSeriesQuery(database.DB, userID).
		Preload("DraftRevision.Image").
		Preload("DraftRevision.I18n").
		Preload("PublishedRevision.Image").
		Preload("PublishedRevision.I18n").
		Preload("Items", func(db *gorm.DB) *gorm.DB {
			return userArtworksQuery(db, userID).Order("sort_index ASC")
		}).
		Preload("Items.DraftRevision.Image").
		Preload("Items.DraftRevision.I18n").
		Preload("Items.PublishedRevision.Image").
		Preload("Items.PublishedRevision.I18n").
		Order("created_at DESC").
		Find(&series).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load works"})
		return
	}

	out := WorksJSONDTO{Series: make([]SerieDTO, 0, len(series))}
	for _, s := range series {
		if view == "published" {
			out.Series = append(out.Series, toSerieDTO_PublishedView(s))
		} else {
			out.Series = append(out.Series, toSerieDTO_DraftView(s))
		}
	}

	c.JSON(http.StatusOK, out)
}

func GetTemplateWorksJSON(c *gin.Context) {
	var series []works.Series

	err := templateSeriesQuery(database.DB).
		Where("published_revision_id IS NOT NULL").
		Preload("PublishedRevision.Image").
		Preload("PublishedRevision.I18n").
		Preload("Items", func(db *gorm.DB) *gorm.DB {
			return templateArtworksQuery(db).
				Where("published_revision_id IS NOT NULL").
				Order("sort_index ASC")
		}).
		Preload("Items.PublishedRevision.Image").
		Preload("Items.PublishedRevision.I18n").
		Order("created_at DESC").
		Find(&series).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load templates"})
		return
	}

	out := WorksJSONDTO{Series: make([]SerieDTO, 0, len(series))}
	for _, s := range series {
		out.Series = append(out.Series, toSerieDTO_PublishedView(s))
	}
	c.JSON(http.StatusOK, out)
}

// ------------------------------
// POST /series  (creates USER series only)
// ------------------------------
func CreateSeries(c *gin.Context) {
	var req CreateSeriesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, ok := mustUserID(c)
	if !ok {
		return
	}
	uid := userID

	err := database.DB.Transaction(func(tx *gorm.DB) error {

		s := works.Series{
			OwnerType: works.OwnerUser,
			UserID:    &uid,
			IDLocked:  req.IDLocked,
		}
		if err := tx.Create(&s).Error; err != nil {
			return err
		}

		dr := works.SeriesRevision{SeriesID: s.ID}
		if req.Image != nil {
			imgID, err := upsertImage(tx, nil, req.Image.OriginalPath, req.Image.WebpPath, req.Image.AvifPath)
			if err != nil {
				return err
			}
			dr.ImageID = imgID
		}
		if err := tx.Create(&dr).Error; err != nil {
			return err
		}

		for lang, v := range req.I18n {
			row := works.SeriesI18nRevision{
				SeriesRevisionID: dr.ID,
				Lang:             lang,
				Title:            v.Title,
				DescriptionSerie: v.DescriptionSerie,
				Year:             v.Year,
			}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		}

		// point draft at it
		if err := tx.Model(&works.Series{}).Where("id = ?", s.ID).Update("draft_revision_id", dr.ID).Error; err != nil {
			return err
		}

		c.JSON(http.StatusCreated, gin.H{"id": s.ID})
		return nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create series", "details": err.Error()})
	}
}

// ------------------------------
// PUT /series/:id (USER series only)
// ------------------------------
func UpdateSeries(c *gin.Context) {
	id := c.Param("id")

	var req UpdateSeriesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, ok := mustUserID(c)
	if !ok {
		return
	}

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		var s works.Series
		if err := tx.First(&s, "id = ? AND owner_type = ? AND user_id = ?", id, works.OwnerUser, userID).Error; err != nil {
			return err
		}
		if s.IDLocked {
			return fmt.Errorf("locked")
		}

		// create or load draft revision
		dr, err := ensureDraftSeriesRevision(tx, &s)
		if err != nil {
			return err
		}

		// update image on draft revision
		if req.Image != nil {
			imgID, err := upsertImage(tx, dr.ImageID, req.Image.OriginalPath, req.Image.WebpPath, req.Image.AvifPath)
			if err != nil {
				return err
			}

			if err := tx.Model(&works.SeriesRevision{}).
				Where("id = ?", dr.ID).
				Update("image_id", imgID).Error; err != nil {
				return err
			}
		}

		// upsert i18n into SeriesI18nRevision using dr.ID
		if req.I18n != nil {
			for lang, v := range req.I18n {
				var existing works.SeriesI18nRevision
				e := tx.First(&existing, "series_revision_id = ? AND lang = ?", dr.ID, lang).Error
				if e != nil {
					if e == gorm.ErrRecordNotFound {
						row := works.SeriesI18nRevision{
							SeriesRevisionID: dr.ID, Lang: lang,
							Title: v.Title, DescriptionSerie: v.DescriptionSerie, Year: v.Year,
						}
						if err := tx.Create(&row).Error; err != nil {
							return err
						}
					} else {
						return e
					}
				} else {
					if err := tx.Model(&works.SeriesI18nRevision{}).
						Where("series_revision_id = ? AND lang = ?", dr.ID, lang).
						Updates(map[string]interface{}{
							"title":             v.Title,
							"description_serie": v.DescriptionSerie,
							"year":              v.Year,
						}).Error; err != nil {
						return err
					}
				}
			}
		}

		c.JSON(http.StatusOK, gin.H{"status": "ok"})
		return nil
	})

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Series not found"})
			return
		}
		if err.Error() == "locked" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Series is locked"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update series", "details": err.Error()})
	}
}

// ------------------------------
// DELETE /series/:id (USER series only)
// ------------------------------
func DeleteSeries(c *gin.Context) {
	id := c.Param("id")

	userID, ok := mustUserID(c)
	if !ok {
		return
	}

	res := database.DB.Delete(&works.Series{}, "id = ? AND owner_type = ? AND user_id = ?", id, works.OwnerUser, userID)
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete series"})
		return
	}
	if res.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Series not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

func PublishSeries(c *gin.Context) {
	id := c.Param("id")
	userID, ok := mustUserID(c)
	if !ok {
		return
	}

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		var s works.Series
		if err := tx.First(&s,
			"id = ? AND owner_type = ? AND user_id = ?",
			id, works.OwnerUser, userID,
		).Error; err != nil {
			return err
		}
		if s.IDLocked {
			return fmt.Errorf("locked")
		}

		// must have draft to publish; if none, create draft cloning published
		dr, err := ensureDraftSeriesRevision(tx, &s)
		if err != nil {
			return err
		}

		// ✅ update SERIES table (not artworks)
		return tx.Model(&works.Series{}).
			Where("id = ? AND owner_type = ? AND user_id = ?", s.ID, works.OwnerUser, userID).
			Updates(map[string]interface{}{
				"published_revision_id": dr.ID,
				"draft_revision_id":     nil,
			}).Error
	})

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Series not found"})
			return
		}
		if err.Error() == "locked" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Series is locked"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to publish series",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "published"})
}

func UnpublishSeries(c *gin.Context) {
	id := c.Param("id")
	userID, ok := mustUserID(c)
	if !ok {
		return
	}

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		var s works.Series
		if err := tx.First(&s,
			"id = ? AND owner_type = ? AND user_id = ?",
			id, works.OwnerUser, userID,
		).Error; err != nil {
			return err
		}

		// If we’re about to remove the published pointer, keep content as draft
		if s.DraftRevisionID == nil && s.PublishedRevisionID != nil {
			if err := tx.Model(&works.Series{}).
				Where("id = ? AND owner_type = ? AND user_id = ?", id, works.OwnerUser, userID).
				Updates(map[string]interface{}{
					"draft_revision_id":     s.PublishedRevisionID,
					"published_revision_id": nil,
				}).Error; err != nil {
				return err
			}
		} else {
			// draft already exists → just unpublish
			if err := tx.Model(&works.Series{}).
				Where("id = ? AND owner_type = ? AND user_id = ?", id, works.OwnerUser, userID).
				Update("published_revision_id", nil).Error; err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Series not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "unpublished"})
}

// ------------------------------
// POST /series/:id/artworks (USER series only)
// ------------------------------
func CreateArtwork(c *gin.Context) {
	seriesID := c.Param("id")

	var req CreateArtworkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, ok := mustUserID(c)
	if !ok {
		return
	}
	uid := userID

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		// ensure USER series exists
		var s works.Series
		if err := tx.First(&s, "id = ? AND owner_type = ? AND user_id = ?", seriesID, works.OwnerUser, userID).Error; err != nil {
			return err
		}

		sortIndex := 0
		if req.SortIndex != nil {
			sortIndex = *req.SortIndex
		}

		// 1) create artwork identity
		a := works.Artwork{
			OwnerType: works.OwnerUser,
			UserID:    &uid,
			SeriesID:  s.ID,
			SortIndex: sortIndex,
			IDLocked:  req.IDLocked,
			Sold:      req.Sold, // stays on identity
		}
		if err := tx.Create(&a).Error; err != nil {
			return err
		}

		// 2) create draft revision with fields
		dr := works.ArtworkRevision{
			ArtworkID: a.ID,
			Year:      req.Year,
			Medium:    req.Medium,
			SizeCM:    req.SizeCM,
			Price:     req.Price,
		}

		if req.Image != nil {
			imgID, err := upsertImage(tx, nil, req.Image.OriginalPath, req.Image.WebpPath, req.Image.AvifPath)
			if err != nil {
				return err
			}
			dr.ImageID = imgID
		}

		if err := tx.Create(&dr).Error; err != nil {
			return err
		}

		for lang, v := range req.I18n {
			row := works.ArtworkI18nRevision{
				ArtworkRevisionID: dr.ID,
				Lang:              lang,
				Title:             v.Title,
				Description:       v.Description,
				Notes:             v.Notes,
			}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		}

		// 3) point identity to draft revision
		if err := tx.Model(&works.Artwork{}).
			Where("id = ?", a.ID).
			Update("draft_revision_id", dr.ID).Error; err != nil {
			return err
		}

		c.JSON(http.StatusCreated, gin.H{"id": a.ID})
		return nil
	})

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Series not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create artwork", "details": err.Error()})
	}
}

// ------------------------------
// PUT /artworks/:id (USER artwork only)
// ------------------------------
func UpdateArtwork(c *gin.Context) {
	id := c.Param("id")

	var req UpdateArtworkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, ok := mustUserID(c)
	if !ok {
		return
	}

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		var a works.Artwork
		if err := tx.First(&a, "id = ? AND owner_type = ? AND user_id = ?", id, works.OwnerUser, userID).Error; err != nil {
			return err
		}
		if a.IDLocked {
			return fmt.Errorf("locked")
		}

		// identity updates (apply to both draft+published views in this phase)
		identityUpdates := map[string]interface{}{}
		if req.SortIndex != nil {
			identityUpdates["sort_index"] = *req.SortIndex
		}
		if req.IDLocked != nil {
			identityUpdates["id_locked"] = *req.IDLocked
		}
		if req.Sold != nil {
			identityUpdates["sold"] = *req.Sold
		}
		if len(identityUpdates) > 0 {
			if err := tx.Model(&works.Artwork{}).
				Where("id = ? AND owner_type = ? AND user_id = ?", a.ID, works.OwnerUser, userID).
				Updates(identityUpdates).Error; err != nil {
				return err
			}
		}

		// draft revision: create/load (clone published if needed)
		dr, err := ensureDraftArtworkRevision(tx, &a)
		fmt.Println("artwork", a.ID, "pub", a.PublishedRevisionID, "draft", a.DraftRevisionID, "dr", dr.ID)

		if err != nil {
			return err
		}

		// revision scalar field updates
		revUpdates := map[string]interface{}{}
		if req.Year != nil {
			revUpdates["year"] = *req.Year
		}
		if req.Medium != nil {
			revUpdates["medium"] = *req.Medium
		}
		if req.SizeCM != nil {
			revUpdates["size_cm"] = *req.SizeCM
		}
		if req.Price != nil {
			revUpdates["price"] = *req.Price
		}
		if len(revUpdates) > 0 {
			if err := tx.Model(&works.ArtworkRevision{}).
				Where("id = ?", dr.ID).
				Updates(revUpdates).Error; err != nil {
				return err
			}
		}

		// image upsert (on revision)
		if req.Image != nil {
			imgID, err := upsertImage(tx, dr.ImageID, req.Image.OriginalPath, req.Image.WebpPath, req.Image.AvifPath)
			if err != nil {
				return err
			}
			if err := tx.Model(&works.ArtworkRevision{}).
				Where("id = ?", dr.ID).
				Update("image_id", imgID).Error; err != nil {
				return err
			}
		}

		// i18n upsert (on revision)
		if req.I18n != nil {
			for lang, v := range req.I18n {
				var existing works.ArtworkI18nRevision
				e := tx.First(&existing, "artwork_revision_id = ? AND lang = ?", dr.ID, lang).Error
				if e != nil {
					if e == gorm.ErrRecordNotFound {
						row := works.ArtworkI18nRevision{
							ArtworkRevisionID: dr.ID,
							Lang:              lang,
							Title:             v.Title,
							Description:       v.Description,
							Notes:             v.Notes,
						}
						if err := tx.Create(&row).Error; err != nil {
							return err
						}
					} else {
						return e
					}
				} else {
					if err := tx.Model(&works.ArtworkI18nRevision{}).
						Where("artwork_revision_id = ? AND lang = ?", dr.ID, lang).
						Updates(map[string]interface{}{
							"title":       v.Title,
							"description": v.Description,
							"notes":       v.Notes,
						}).Error; err != nil {
						return err
					}
				}
			}
		}

		c.JSON(http.StatusOK, gin.H{"status": "ok"})
		return nil
	})

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Artwork not found"})
			return
		}
		if err.Error() == "locked" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Artwork is locked"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update artwork", "details": err.Error()})
	}
}

// ------------------------------
// DELETE /artworks/:id (USER artwork only)
// ------------------------------
func DeleteArtwork(c *gin.Context) {
	id := c.Param("id")

	userID, ok := mustUserID(c)
	if !ok {
		return
	}

	res := database.DB.Delete(&works.Artwork{}, "id = ? AND owner_type = ? AND user_id = ?", id, works.OwnerUser, userID)
	if res.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete artwork"})
		return
	}
	if res.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Artwork not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

func PublishArtwork(c *gin.Context) {
	id := c.Param("id")
	userID, ok := mustUserID(c)
	if !ok {
		return
	}

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		var a works.Artwork
		if err := tx.First(&a, "id = ? AND owner_type = ? AND user_id = ?", id, works.OwnerUser, userID).Error; err != nil {
			return err
		}
		if a.IDLocked {
			return fmt.Errorf("locked")
		}

		// must have draft to publish; if none, create draft cloning published
		dr, err := ensureDraftArtworkRevision(tx, &a)
		if err != nil {
			return err
		}

		// publish = point published_revision_id to draft revision
		return tx.Model(&works.Artwork{}).
			Where("id = ? AND owner_type = ? AND user_id = ?", a.ID, works.OwnerUser, userID).
			Updates(map[string]interface{}{
				"published_revision_id": dr.ID,
				"draft_revision_id":     nil, // ✅ important
			}).Error
	})

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Artwork not found"})
			return
		}
		if err.Error() == "locked" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Artwork is locked"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to publish artwork", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "published"})
}

func UnpublishArtwork(c *gin.Context) {
	id := c.Param("id")
	userID, ok := mustUserID(c)
	if !ok {
		return
	}

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		var a works.Artwork
		if err := tx.First(&a,
			"id = ? AND owner_type = ? AND user_id = ?",
			id, works.OwnerUser, userID,
		).Error; err != nil {
			return err
		}

		if a.DraftRevisionID == nil && a.PublishedRevisionID != nil {
			return tx.Model(&works.Artwork{}).
				Where("id = ? AND owner_type = ? AND user_id = ?", id, works.OwnerUser, userID).
				Updates(map[string]interface{}{
					"draft_revision_id":     a.PublishedRevisionID,
					"published_revision_id": nil,
				}).Error
		}

		return tx.Model(&works.Artwork{}).
			Where("id = ? AND owner_type = ? AND user_id = ?", id, works.OwnerUser, userID).
			Update("published_revision_id", nil).Error
	})

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Artwork not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "unpublished"})
}

// ------------------------------
// PUT /series/:id/artworks/reorder (USER series only)
// ------------------------------
func ReorderArtworks(c *gin.Context) {
	seriesID := c.Param("id")

	userID, ok := mustUserID(c)
	if !ok {
		return
	}

	var req ReorderArtworksRequest
	if err := c.ShouldBindJSON(&req); err != nil || len(req.ArtworkIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "artwork_ids required"})
		return
	}

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		// Ensure USER series
		var s works.Series
		if err := tx.First(&s, "id = ? AND owner_type = ? AND user_id = ?", seriesID, works.OwnerUser, userID).Error; err != nil {
			return err
		}
		if s.IDLocked {
			return fmt.Errorf("locked")
		}

		for i, artworkID := range req.ArtworkIDs {
			if err := tx.Model(&works.Artwork{}).
				Where("id = ? AND series_id = ? AND owner_type = ? AND user_id = ?", artworkID, s.ID, works.OwnerUser, userID).
				Update("sort_index", i).Error; err != nil {
				return err
			}
		}

		c.JSON(http.StatusOK, gin.H{"status": "ok"})
		return nil
	})

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Series not found"})
			return
		}
		if err.Error() == "locked" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Series is locked"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reorder artworks", "details": err.Error()})
	}
}

// ------------------------------
// GET /series/:id (USER, draft view: draft -> fallback to published)
// ------------------------------
func GetSeriesByID(c *gin.Context) {
	id := c.Param("id")

	userID, ok := mustUserID(c)
	if !ok {
		return
	}

	var s works.Series
	err := database.DB.
		// series revisions
		Preload("DraftRevision.Image").
		Preload("DraftRevision.I18n").
		Preload("PublishedRevision.Image").
		Preload("PublishedRevision.I18n").
		// items identities
		Preload("Items", func(db *gorm.DB) *gorm.DB {
			return db.Where("owner_type = ? AND user_id = ?", works.OwnerUser, userID).
				Order("sort_index ASC")
		}).
		// item revisions
		Preload("Items.DraftRevision.Image").
		Preload("Items.DraftRevision.I18n").
		Preload("Items.PublishedRevision.Image").
		Preload("Items.PublishedRevision.I18n").
		First(&s, "id = ? AND owner_type = ? AND user_id = ?", id, works.OwnerUser, userID).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Series not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load series"})
		return
	}

	c.JSON(http.StatusOK, toSerieDTO_DraftView(s))
}

// ------------------------------
// GET /artworks/:id (safe default: USER only)
// ------------------------------
func GetArtworkByID(c *gin.Context) {
	id := c.Param("id")

	userID, ok := mustUserID(c)
	if !ok {
		return
	}

	view := c.DefaultQuery("view", "draft") // "draft" | "published"

	var a works.Artwork
	err := database.DB.
		Preload("DraftRevision.Image").
		Preload("DraftRevision.I18n").
		Preload("PublishedRevision.Image").
		Preload("PublishedRevision.I18n").
		First(&a, "id = ? AND owner_type = ? AND user_id = ?", id, works.OwnerUser, userID).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Artwork not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to load artwork"})
		return
	}

	// choose revision based on view
	var rev *works.ArtworkRevision
	if view == "published" {
		rev = a.PublishedRevision
		if rev == nil {
			// published view requested but not published
			c.JSON(http.StatusOK, toArtworkDTOFromRevision(a, nil))
			return
		}
	} else {
		// draft view: draft if exists, else published
		rev = a.DraftRevision
		if rev == nil {
			rev = a.PublishedRevision
		}
	}

	c.JSON(http.StatusOK, toArtworkDTOFromRevision(a, rev))
}

func CreateTemplateSeries(c *gin.Context) {
	var req CreateSeriesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		// 1) create series identity (system-owned)
		s := works.Series{
			OwnerType: works.OwnerSystem,
			UserID:    nil,
			IDLocked:  req.IDLocked,
		}
		if err := tx.Create(&s).Error; err != nil {
			return err
		}

		// 2) create draft revision
		dr := works.SeriesRevision{
			SeriesID: s.ID,
		}

		// image goes on revision
		if req.Image != nil {
			imgID, err := upsertImage(tx, nil, req.Image.OriginalPath, req.Image.WebpPath, req.Image.AvifPath)
			if err != nil {
				return err
			}
			dr.ImageID = imgID
		}

		if err := tx.Create(&dr).Error; err != nil {
			return err
		}

		// 3) i18n goes on revision i18n table
		for lang, v := range req.I18n {
			row := works.SeriesI18nRevision{
				SeriesRevisionID: dr.ID,
				Lang:             lang,
				Title:            v.Title,
				DescriptionSerie: v.DescriptionSerie,
				Year:             v.Year,
			}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		}

		// 4) point identity to revision(s)
		updates := map[string]interface{}{
			"draft_revision_id": dr.ID,
			// recommended for templates so they appear in template listing:
			"published_revision_id": dr.ID,
		}

		if err := tx.Model(&works.Series{}).
			Where("id = ? AND owner_type = ?", s.ID, works.OwnerSystem).
			Updates(updates).Error; err != nil {
			return err
		}

		c.JSON(http.StatusCreated, gin.H{"id": s.ID})
		return nil
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create template series", "details": err.Error()})
	}
}

func CreateTemplateArtwork(c *gin.Context) {
	seriesID := c.Param("id")

	var req CreateArtworkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		// ensure template series exists
		var s works.Series
		if err := tx.First(&s, "id = ? AND owner_type = ?", seriesID, works.OwnerSystem).Error; err != nil {
			return err
		}

		sortIndex := 0
		if req.SortIndex != nil {
			sortIndex = *req.SortIndex
		}

		// 1) create artwork identity (system-owned)
		a := works.Artwork{
			OwnerType: works.OwnerSystem,
			UserID:    nil,
			SeriesID:  s.ID,
			SortIndex: sortIndex,
			IDLocked:  req.IDLocked,
			Sold:      req.Sold, // up to you; templates usually false
		}
		if err := tx.Create(&a).Error; err != nil {
			return err
		}

		// 2) create draft revision with fields
		dr := works.ArtworkRevision{
			ArtworkID: a.ID,
			Year:      req.Year,
			Medium:    req.Medium,
			SizeCM:    req.SizeCM,
			Price:     req.Price,
		}

		if req.Image != nil {
			imgID, err := upsertImage(tx, nil, req.Image.OriginalPath, req.Image.WebpPath, req.Image.AvifPath)
			if err != nil {
				return err
			}
			dr.ImageID = imgID
		}

		if err := tx.Create(&dr).Error; err != nil {
			return err
		}

		for lang, v := range req.I18n {
			row := works.ArtworkI18nRevision{
				ArtworkRevisionID: dr.ID,
				Lang:              lang,
				Title:             v.Title,
				Description:       v.Description,
				Notes:             v.Notes,
			}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		}

		// 3) point identity to draft revision (and optionally published)
		updates := map[string]interface{}{
			"draft_revision_id": dr.ID,
			// recommended for templates:
			"published_revision_id": dr.ID,
		}

		if err := tx.Model(&works.Artwork{}).
			Where("id = ? AND owner_type = ?", a.ID, works.OwnerSystem).
			Updates(updates).Error; err != nil {
			return err
		}

		c.JSON(http.StatusCreated, gin.H{"id": a.ID})
		return nil
	})

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Template series not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create template artwork", "details": err.Error()})
	}
}

// DELETE /series/:id/artworks
func DeleteAllArtworksOfSeries(c *gin.Context) {
	seriesID := c.Param("id")

	userID, ok := mustUserID(c)
	if !ok {
		return
	}

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		// ensure USER series exists
		var s works.Series
		if err := tx.First(&s,
			"id = ? AND owner_type = ? AND user_id = ?",
			seriesID, works.OwnerUser, userID,
		).Error; err != nil {
			return err
		}
		if s.IDLocked {
			return fmt.Errorf("locked")
		}

		// load artworks (only fields we need)
		var items []works.Artwork
		if err := tx.Select("id", "draft_revision_id", "published_revision_id").
			Where("series_id = ? AND owner_type = ? AND user_id = ?", s.ID, works.OwnerUser, userID).
			Find(&items).Error; err != nil {
			return err
		}

		if len(items) == 0 {
			c.JSON(http.StatusOK, gin.H{"status": "deleted", "count": 0})
			return nil
		}

		artworkIDs := make([]string, 0, len(items))
		revisionIDs := make([]string, 0, len(items)*2)

		// collect ids
		for _, a := range items {
			artworkIDs = append(artworkIDs, a.ID)
			if a.DraftRevisionID != nil && *a.DraftRevisionID != "" {
				revisionIDs = append(revisionIDs, *a.DraftRevisionID)
			}
			if a.PublishedRevisionID != nil && *a.PublishedRevisionID != "" {
				revisionIDs = append(revisionIDs, *a.PublishedRevisionID)
			}
		}

		// delete i18n rows for those revisions
		if len(revisionIDs) > 0 {
			if err := tx.Where("artwork_revision_id IN ?", revisionIDs).
				Delete(&works.ArtworkI18nRevision{}).Error; err != nil {
				return err
			}

			// delete revision rows
			if err := tx.Where("id IN ?", revisionIDs).
				Delete(&works.ArtworkRevision{}).Error; err != nil {
				return err
			}
		}

		// delete artwork identity rows
		res := tx.Where("id IN ?", artworkIDs).Delete(&works.Artwork{})
		if res.Error != nil {
			return res.Error
		}

		c.JSON(http.StatusOK, gin.H{
			"status": "deleted",
			"count":  res.RowsAffected,
		})
		return nil
	})

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Series not found"})
			return
		}
		if err.Error() == "locked" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Series is locked"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed", "details": err.Error()})
	}
}

// ------------------------------
// POST /series/:id/artworks/discard-drafts
// body: { "artwork_ids": ["...","..."] }   (empty => discard ALL in series)
// Discard = remove draft pointer so draft-view falls back to published.
// If an artwork is unpublished (published_revision_id NULL), we SKIP it (to avoid wiping content).
// ------------------------------

type BulkDiscardArtworkDraftsRequest struct {
	ArtworkIDs []string `json:"artwork_ids"`
}

func BulkDiscardArtworkDrafts(c *gin.Context) {
	seriesID := c.Param("id")

	userID, ok := mustUserID(c)
	if !ok {
		return
	}

	var req BulkDiscardArtworkDraftsRequest
	_ = c.ShouldBindJSON(&req) // allow empty body

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		// ensure USER series exists
		var s works.Series
		if err := tx.First(&s,
			"id = ? AND owner_type = ? AND user_id = ?",
			seriesID, works.OwnerUser, userID,
		).Error; err != nil {
			return err
		}
		if s.IDLocked {
			return fmt.Errorf("locked")
		}

		q := tx.Model(&works.Artwork{}).
			Select("id", "draft_revision_id", "published_revision_id").
			Where("series_id = ? AND owner_type = ? AND user_id = ?", s.ID, works.OwnerUser, userID)

		// If artwork_ids provided -> only those. If empty -> all artworks in series.
		if len(req.ArtworkIDs) > 0 {
			q = q.Where("id IN ?", req.ArtworkIDs)
		}

		var items []works.Artwork
		if err := q.Find(&items).Error; err != nil {
			return err
		}

		if len(items) == 0 {
			c.JSON(http.StatusOK, gin.H{"status": "ok", "updated": 0, "skipped": 0})
			return nil
		}

		updateIDs := make([]string, 0, len(items))
		orphanDraftRevIDs := make([]string, 0, len(items))
		skipped := 0

		for _, a := range items {
			// CASE (unpublished) -> skip to avoid draft-view becoming empty
			if a.PublishedRevisionID == nil || *a.PublishedRevisionID == "" {
				skipped++
				continue
			}

			// CASE 2: published only (no draft) -> nothing to discard
			if a.DraftRevisionID == nil || *a.DraftRevisionID == "" {
				skipped++
				continue
			}

			// CASE 1: published + draft -> clear draft pointer
			updateIDs = append(updateIDs, a.ID)

			// cleanup: only delete draft revision if it’s different than published
			if *a.DraftRevisionID != *a.PublishedRevisionID {
				orphanDraftRevIDs = append(orphanDraftRevIDs, *a.DraftRevisionID)
			}
		}

		if len(updateIDs) == 0 {
			c.JSON(http.StatusOK, gin.H{"status": "ok", "updated": 0, "skipped": skipped})
			return nil
		}

		// Clear draft pointers
		if err := tx.Model(&works.Artwork{}).
			Where("id IN ? AND owner_type = ? AND user_id = ?", updateIDs, works.OwnerUser, userID).
			Update("draft_revision_id", nil).Error; err != nil {
			return err
		}

		// Optional cleanup of orphan draft revisions (+i18n)
		if len(orphanDraftRevIDs) > 0 {
			if err := tx.Where("artwork_revision_id IN ?", orphanDraftRevIDs).
				Delete(&works.ArtworkI18nRevision{}).Error; err != nil {
				return err
			}
			if err := tx.Where("id IN ?", orphanDraftRevIDs).
				Delete(&works.ArtworkRevision{}).Error; err != nil {
				return err
			}
		}

		c.JSON(http.StatusOK, gin.H{
			"status":  "ok",
			"updated": len(updateIDs),
			"skipped": skipped,
		})
		return nil
	})

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Series not found"})
			return
		}
		if err.Error() == "locked" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Series is locked"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed", "details": err.Error()})
	}
}
