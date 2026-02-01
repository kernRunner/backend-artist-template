package works

import (
	"registration-app/internal/domain/media"
	"registration-app/internal/domain/works"
)

type ImageRefDTO struct {
	Original string `json:"original"`
	Webp     string `json:"webp"`
	Avif     string `json:"avif"`
}

type RevisionMetaDTO struct {
	View                string  `json:"view"` // "draft" | "published" | "empty"
	Published           bool    `json:"published"`
	HasDraft            bool    `json:"hasDraft"`
	DraftRevisionID     *string `json:"draftRevisionId,omitempty"`
	PublishedRevisionID *string `json:"publishedRevisionId,omitempty"`
}

type ArtworkItemDTO struct {
	ID       string       `json:"id"`
	IDLocked bool         `json:"idLocked,omitempty"`
	Sold     bool         `json:"sold"`
	Image    *ImageRefDTO `json:"image,omitempty"`

	Meta RevisionMetaDTO `json:"meta"`

	I18n map[string]map[string]string `json:"i18n"`

	Year   string `json:"year"`
	Medium string `json:"medium"`
	SizeCM string `json:"size_cm"`
	Price  string `json:"price"`
}

type SerieDTO struct {
	ID       string       `json:"id"`
	IDLocked bool         `json:"idLocked,omitempty"`
	Image    *ImageRefDTO `json:"image,omitempty"`

	Meta RevisionMetaDTO `json:"meta"`

	I18n  map[string]map[string]string `json:"i18n"`
	Items []ArtworkItemDTO             `json:"items"`
}

type WorksJSONDTO struct {
	Series []SerieDTO `json:"series"`
}

func toImageRefDTO(img *media.Image) *ImageRefDTO {
	if img == nil {
		return nil
	}
	webp := ""
	avif := ""
	if img.WebpPath != nil {
		webp = *img.WebpPath
	}
	if img.AvifPath != nil {
		avif = *img.AvifPath
	}
	return &ImageRefDTO{
		Original: img.OriginalPath,
		Webp:     webp,
		Avif:     avif,
	}
}

func pickSeriesRevisionDraftView(s works.Series) *works.SeriesRevision {
	if s.DraftRevision != nil {
		return s.DraftRevision
	}
	return s.PublishedRevision
}

func pickArtworkRevisionDraftView(a works.Artwork) *works.ArtworkRevision {
	if a.DraftRevision != nil {
		return a.DraftRevision
	}
	return a.PublishedRevision
}

func toArtworkDTOFromRevision(a works.Artwork, rev *works.ArtworkRevision) ArtworkItemDTO {
	i18n := map[string]map[string]string{}
	if rev != nil {
		for _, t := range rev.I18n {
			i18n[t.Lang] = map[string]string{
				"title":       t.Title,
				"description": t.Description,
				"notes":       t.Notes,
			}
		}
	}

	dto := ArtworkItemDTO{
		ID:   a.ID,
		Sold: a.Sold,
		I18n: i18n,
		Meta: artworkMeta(a),
	}

	if a.IDLocked {
		dto.IDLocked = true
	}
	if rev != nil {
		dto.Image = toImageRefDTO(rev.Image)
		dto.Year = rev.Year
		dto.Medium = rev.Medium
		dto.SizeCM = rev.SizeCM
		dto.Price = rev.Price
	}

	return dto
}

// Draft view: use draft if exists, else published
func toSerieDTO_DraftView(s works.Series) SerieDTO {
	rev := pickSeriesRevisionDraftView(s)

	i18n := map[string]map[string]string{}
	if rev != nil {
		for _, t := range rev.I18n {
			i18n[t.Lang] = map[string]string{
				"title":            t.Title,
				"descriptionSerie": t.DescriptionSerie,
				"year":             t.Year,
			}
		}
	}

	items := make([]ArtworkItemDTO, 0, len(s.Items))
	for _, a := range s.Items {
		arev := pickArtworkRevisionDraftView(a)
		items = append(items, toArtworkDTOFromRevision(a, arev))
	}

	dto := SerieDTO{
		ID:    s.ID,
		I18n:  i18n,
		Items: items,
		Meta:  seriesMeta(s),
	}

	if s.IDLocked {
		dto.IDLocked = true
	}
	if rev != nil {
		dto.Image = toImageRefDTO(rev.Image)
	}

	return dto
}

// Published view: only use published revision
func toSerieDTO_PublishedView(s works.Series) SerieDTO {
	rev := s.PublishedRevision

	i18n := map[string]map[string]string{}
	if rev != nil {
		for _, t := range rev.I18n {
			i18n[t.Lang] = map[string]string{
				"title":            t.Title,
				"descriptionSerie": t.DescriptionSerie,
				"year":             t.Year,
			}
		}
	}

	items := make([]ArtworkItemDTO, 0, len(s.Items))
	for _, a := range s.Items {
		// only published fields
		items = append(items, toArtworkDTOFromRevision(a, a.PublishedRevision))
	}

	dto := SerieDTO{
		ID:    s.ID,
		I18n:  i18n,
		Items: items,
	}

	if s.IDLocked {
		dto.IDLocked = true
	}

	if rev != nil {
		dto.Image = toImageRefDTO(rev.Image)
	}

	return dto
}

func seriesMeta(s works.Series) RevisionMetaDTO {
	hasPub := s.PublishedRevisionID != nil && *s.PublishedRevisionID != ""
	hasDraft := s.DraftRevisionID != nil && *s.DraftRevisionID != ""

	if hasDraft && hasPub && *s.DraftRevisionID == *s.PublishedRevisionID {
		hasDraft = false
	}

	view := "empty"
	if hasDraft {
		view = "draft"
	} else if hasPub {
		view = "published"
	}

	return RevisionMetaDTO{
		View:                view,
		Published:           hasPub,
		HasDraft:            hasDraft,
		DraftRevisionID:     s.DraftRevisionID,
		PublishedRevisionID: s.PublishedRevisionID,
	}
}

func artworkMeta(a works.Artwork) RevisionMetaDTO {
	hasPub := a.PublishedRevisionID != nil && *a.PublishedRevisionID != ""

	hasDraft := a.DraftRevisionID != nil && *a.DraftRevisionID != ""
	// if draft == published, treat as no draft
	if hasDraft && hasPub && *a.DraftRevisionID == *a.PublishedRevisionID {
		hasDraft = false
	}

	view := "empty"
	if hasDraft {
		view = "draft"
	} else if hasPub {
		view = "published"
	}

	return RevisionMetaDTO{
		View:                view,
		Published:           hasPub,
		HasDraft:            hasDraft,
		DraftRevisionID:     a.DraftRevisionID,
		PublishedRevisionID: a.PublishedRevisionID,
	}
}
