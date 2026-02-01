package site

type LimitedRules struct {
	MaxArtworks          int
	HideGalleries        bool
	NoIndex              bool
	ShowPlatformBranding bool
}

// Keep this in sync with access states by using plain strings.
// "limited" and "locked" are the only ones that matter for site limits.
func LimitedRulesFor(accessState string) *LimitedRules {
	if accessState != "limited" && accessState != "locked" {
		return nil
	}

	return &LimitedRules{
		MaxArtworks:          3,
		HideGalleries:        true,
		NoIndex:              true,
		ShowPlatformBranding: true,
	}
}
