package site

import (
	"fmt"
	"regexp"
	"strings"

	"registration-app/internal/domain/users"

	"gorm.io/gorm"
)

/*
	Site / slug helpers
	-------------------
	- Responsible ONLY for:
	  • generating slugs
	  • persisting them
	  • building public URLs
	- No access logic, no billing logic here
*/

var (
	nonSlug   = regexp.MustCompile(`[^a-z0-9\-]+`)
	multiDash = regexp.MustCompile(`-+`)
)

// MakeSlug generates a URL-safe base slug from user name.
// Example: "John Doe" -> "john-doe"
func MakeSlug(name, lastname string) string {
	base := strings.ToLower(strings.TrimSpace(name + " " + lastname))
	base = strings.ReplaceAll(base, " ", "-")
	base = nonSlug.ReplaceAllString(base, "")
	base = multiDash.ReplaceAllString(base, "-")
	base = strings.Trim(base, "-")

	if base == "" {
		base = "user"
	}
	return base
}

// EnsureSiteSlug ensures user.SiteSlug exists and is persisted.
// Must be called AFTER user has an ID (after Create).
//
// IMPORTANT: pass db in, do NOT import registration-app/database here (avoids import cycle).
func EnsureSiteSlug(db *gorm.DB, user *users.User) (string, error) {
	if user == nil {
		return "", fmt.Errorf("user is nil")
	}
	if db == nil {
		return "", fmt.Errorf("db is nil")
	}

	// Already exists
	if user.SiteSlug != nil && strings.TrimSpace(*user.SiteSlug) != "" {
		return strings.TrimSpace(*user.SiteSlug), nil
	}

	if user.ID == 0 {
		return "", fmt.Errorf("user ID missing (call EnsureSiteSlug after Create)")
	}

	base := MakeSlug(user.Name, user.Lastname)
	slug := fmt.Sprintf("%s-%d", base, user.ID)

	// Update struct
	user.SiteSlug = &slug

	// Persist ONLY the slug column
	if err := db.
		Model(&users.User{}).
		Where("id = ?", user.ID).
		Update("site_slug", slug).Error; err != nil {
		return "", err
	}

	return slug, nil
}

// BuildPublicURL builds the public site URL from a slug.
// Example: "john-doe-32" -> "https://john-doe-32.yourplatform.com"
func BuildPublicURL(slug string) string {
	return "https://" + slug + ".yourplatform.com"
}
