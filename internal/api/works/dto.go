package works

type LangString string

// ---------- requests

type ImageInput struct {
	OriginalPath string  `json:"original_path" binding:"required"`
	WebpPath     *string `json:"webp_path"`
	AvifPath     *string `json:"avif_path"`
}

type SeriesI18nInput struct {
	Title            string `json:"title" binding:"required"`
	DescriptionSerie string `json:"description_serie"`
	Year             string `json:"year"`
}

type ArtworkI18nInput struct {
	Title       string `json:"title" binding:"required"`
	Description string `json:"description"`
	Notes       string `json:"notes"`
}

type CreateSeriesRequest struct {
	IDLocked bool                       `json:"id_locked"`
	Image    *ImageInput                `json:"image"`
	I18n     map[string]SeriesI18nInput `json:"i18n" binding:"required"` // { "en": {...}, "de": {...} }
}

type UpdateSeriesRequest struct {
	IDLocked *bool                      `json:"id_locked"`
	Image    *ImageInput                `json:"image"`
	I18n     map[string]SeriesI18nInput `json:"i18n"` // upsert languages
}

type CreateArtworkRequest struct {
	SortIndex *int        `json:"sort_index"`
	IDLocked  bool        `json:"id_locked"`
	Sold      bool        `json:"sold"`
	Image     *ImageInput `json:"image"`

	Year   string `json:"year"`
	Medium string `json:"medium"`
	SizeCM string `json:"size_cm"`
	Price  string `json:"price"`

	I18n map[string]ArtworkI18nInput `json:"i18n" binding:"required"`
}

type UpdateArtworkRequest struct {
	SortIndex *int        `json:"sort_index"`
	IDLocked  *bool       `json:"id_locked"`
	Sold      *bool       `json:"sold"`
	Image     *ImageInput `json:"image"`

	Year   *string `json:"year"`
	Medium *string `json:"medium"`
	SizeCM *string `json:"size_cm"`
	Price  *string `json:"price"`

	I18n map[string]ArtworkI18nInput `json:"i18n"` // upsert languages
}

type ReorderArtworksRequest struct {
	ArtworkIDs []string `json:"artwork_ids" binding:"required"` // ordered list
}

type PublishRequest struct {
	Publish bool `json:"publish" binding:"required"`
}
