package siteapi

import "encoding/json"

type TemplateDTO struct {
	ID   string `json:"id"`
	Slug string `json:"slug"`
	Name string `json:"name"`
}

type BlockDTO struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"`
	SortIndex int             `json:"sortIndex"`
	Props     json.RawMessage `json:"props"`
}

type PageDTO struct {
	ID     string     `json:"id,omitempty"`
	Slug   string     `json:"slug"`
	Lang   string     `json:"lang"`
	Status string     `json:"status"`
	Blocks []BlockDTO `json:"blocks"`
}

type GetTemplatesResponse struct {
	Templates []TemplateDTO `json:"templates"`
}

type GetTemplateResponse struct {
	Template TemplateDTO `json:"template"`
	Pages    []PageDTO   `json:"pages"`
}

type GetUserSiteResponse struct {
	Pages []PageDTO `json:"pages"`
}
