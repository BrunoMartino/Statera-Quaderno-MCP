package wordpress

import "encoding/json"

// WPText is title/content/excerpt as WP REST returns them (object or string).
type WPText struct {
	Raw      string `json:"raw,omitempty"`
	Rendered string `json:"rendered,omitempty"`
}

func (t *WPText) UnmarshalJSON(b []byte) error {
	if len(b) == 0 || string(b) == "null" {
		return nil
	}
	if b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		t.Rendered = s
		return nil
	}
	var obj struct {
		Raw      string `json:"raw"`
		Rendered string `json:"rendered"`
	}
	if err := json.Unmarshal(b, &obj); err != nil {
		return err
	}
	t.Raw = obj.Raw
	t.Rendered = obj.Rendered
	return nil
}

type Post struct {
	ID            int    `json:"id"`
	Type          string `json:"type,omitempty"`
	Link          string `json:"link,omitempty"`
	Title         WPText `json:"title"`
	Content       WPText `json:"content,omitempty"`
	Excerpt       WPText `json:"excerpt,omitempty"`
	Slug          string `json:"slug,omitempty"`
	Status        string `json:"status,omitempty"`
	FeaturedMedia int    `json:"featured_media,omitempty"`
	Modified      string `json:"modified,omitempty"`
}

type PostListItem struct {
	ID     int    `json:"id"`
	Title  WPText `json:"title"`
	Slug   string `json:"slug,omitempty"`
	Status string `json:"status,omitempty"`
	Link   string `json:"link,omitempty"`
}

type PostWrite struct {
	Title         *string `json:"title,omitempty"`
	Content       *string `json:"content,omitempty"`
	Excerpt       *string `json:"excerpt,omitempty"`
	Slug          *string `json:"slug,omitempty"`
	Status        *string `json:"status,omitempty"`
	FeaturedMedia *int    `json:"featured_media,omitempty"`
	Categories    []int   `json:"categories,omitempty"`
	Tags          []int   `json:"tags,omitempty"`
}

type Page struct {
	ID            int    `json:"id"`
	Type          string `json:"type,omitempty"`
	Link          string `json:"link,omitempty"`
	Title         WPText `json:"title"`
	Content       WPText `json:"content,omitempty"`
	Excerpt       WPText `json:"excerpt,omitempty"`
	Slug          string `json:"slug,omitempty"`
	Status        string `json:"status,omitempty"`
	FeaturedMedia int    `json:"featured_media,omitempty"`
	Modified      string `json:"modified,omitempty"`
}

type PageListItem struct {
	ID     int    `json:"id"`
	Title  WPText `json:"title"`
	Slug   string `json:"slug,omitempty"`
	Status string `json:"status,omitempty"`
	Link   string `json:"link,omitempty"`
}

type Media struct {
	ID        int    `json:"id"`
	SourceURL string `json:"source_url,omitempty"`
	MimeType  string `json:"mime_type,omitempty"`
	AltText   string `json:"alt_text,omitempty"`
}
