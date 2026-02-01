package siteapi

import (
	"registration-app/database"
	"registration-app/internal/domain/site"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// GET /templates/site
func ListSiteTemplates(c *gin.Context) {
	var templates []site.Template
	if err := database.DB.
		Where("active = true").
		Order("name ASC").
		Find(&templates).Error; err != nil {
		c.JSON(500, gin.H{"error": "Failed to load templates"})
		return
	}

	out := GetTemplatesResponse{Templates: make([]TemplateDTO, 0, len(templates))}
	for _, t := range templates {
		out.Templates = append(out.Templates, TemplateDTO{ID: t.ID, Slug: t.Slug, Name: t.Name})
	}
	c.JSON(200, out)
}

// GET /templates/site/:slug
func GetSiteTemplate(c *gin.Context) {
	slug := c.Param("slug")

	var tmpl site.Template
	if err := database.DB.First(&tmpl, "slug = ? AND active = true", slug).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(404, gin.H{"error": "Template not found"})
			return
		}
		c.JSON(500, gin.H{"error": "Failed to load template"})
		return
	}

	var pages []site.SitePage
	if err := templatePagesQuery(database.DB, tmpl.ID).
		Preload("Blocks", func(db *gorm.DB) *gorm.DB { return db.Order("sort_index ASC") }).
		Order("slug ASC, lang ASC").
		Find(&pages).Error; err != nil {
		c.JSON(500, gin.H{"error": "Failed to load template pages"})
		return
	}

	resp := GetTemplateResponse{
		Template: TemplateDTO{ID: tmpl.ID, Slug: tmpl.Slug, Name: tmpl.Name},
		Pages:    make([]PageDTO, 0, len(pages)),
	}

	for _, p := range pages {
		dto := PageDTO{
			Slug:   p.Slug,
			Lang:   p.Lang,
			Status: p.Status,
			Blocks: make([]BlockDTO, 0, len(p.Blocks)),
		}
		for _, b := range p.Blocks {
			dto.Blocks = append(dto.Blocks, BlockDTO{
				ID:        b.ID,
				Type:      b.Type,
				SortIndex: b.SortIndex,
				Props:     b.Props, // ✅ already json.RawMessage
			})
		}
		resp.Pages = append(resp.Pages, dto)
	}

	c.JSON(200, resp)
}

// GET /site (auth)
func GetUserSite(c *gin.Context) {
	userID, ok := mustUserID(c)
	if !ok {
		return
	}

	var pages []site.SitePage
	if err := userPagesQuery(database.DB, userID).
		Preload("Blocks", func(db *gorm.DB) *gorm.DB { return db.Order("sort_index ASC") }).
		Order("slug ASC, lang ASC").
		Find(&pages).Error; err != nil {
		c.JSON(500, gin.H{"error": "Failed to load site"})
		return
	}

	out := GetUserSiteResponse{Pages: make([]PageDTO, 0, len(pages))}
	for _, p := range pages {
		page := PageDTO{
			ID:     p.ID,
			Slug:   p.Slug,
			Lang:   p.Lang,
			Status: p.Status,
			Blocks: make([]BlockDTO, 0, len(p.Blocks)),
		}
		for _, b := range p.Blocks {
			page.Blocks = append(page.Blocks, BlockDTO{
				ID:        b.ID,
				Type:      b.Type,
				SortIndex: b.SortIndex,
				Props:     b.Props,
			})
		}
		out.Pages = append(out.Pages, page)
	}

	c.JSON(200, out)
}

// POST /site/from-template/:slug (auth)
func CopySiteFromTemplate(c *gin.Context) {
	userID, ok := mustUserID(c)
	if !ok {
		return
	}

	slug := c.Param("slug")

	var tmpl site.Template
	if err := database.DB.First(&tmpl, "slug = ? AND active = true", slug).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(404, gin.H{"error": "Template not found"})
			return
		}
		c.JSON(500, gin.H{"error": "Failed to load template"})
		return
	}

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		var tPages []site.SitePage
		if err := templatePagesQuery(tx, tmpl.ID).
			Preload("Blocks", func(db *gorm.DB) *gorm.DB { return db.Order("sort_index ASC") }).
			Find(&tPages).Error; err != nil {
			return err
		}

		// prevent duplicates
		var count int64
		if err := userPagesQuery(tx, userID).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			c.JSON(409, gin.H{"error": "User already has site pages"})
			return nil
		}

		createdPages := 0
		createdBlocks := 0

		for _, tp := range tPages {
			uid := userID
			up := site.SitePage{
				OwnerType: site.OwnerUser,
				UserID:    &uid,
				Slug:      tp.Slug,
				Lang:      tp.Lang,
				Status:    "draft",
			}
			if err := tx.Create(&up).Error; err != nil {
				return err
			}
			createdPages++

			for _, tb := range tp.Blocks {
				ub := site.SitePageBlock{
					PageID:    up.ID,
					SortIndex: tb.SortIndex,
					Type:      tb.Type,
					Props:     tb.Props,
				}
				if err := tx.Create(&ub).Error; err != nil {
					return err
				}
				createdBlocks++
			}
		}

		c.JSON(201, gin.H{
			"status":        "ok",
			"template":      tmpl.Slug,
			"createdPages":  createdPages,
			"createdBlocks": createdBlocks,
		})
		return nil
	})

	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to copy template", "details": err.Error()})
	}
}

func mustUserID(c *gin.Context) (uint, bool) {
	v, ok := c.Get("user_id")
	if !ok {
		c.JSON(401, gin.H{"error": "Unauthorized"})
		return 0, false
	}
	uid, ok := v.(uint)
	if !ok || uid == 0 {
		c.JSON(401, gin.H{"error": "Unauthorized"})
		return 0, false
	}
	return uid, true
}
