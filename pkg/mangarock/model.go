package mangarock

import "time"

// Manga contains a manga. This struct is returned by endpoints listing mangas.
type Manga struct {
	ID              string    `json:"oid"`
	Name            string    `json:"name"`
	Author          Author    `json:"-"`
	Authors         []Author  `json:"authors"`
	AuthorIDs       []string  `json:"author_ids"`
	Genres          []string  `json:"genres"`
	Rank            int       `json:"rank"`
	UpdatedChapters int       `json:"updated_chapters"`
	NewChapters     []Chapter `json:"new_chapters"`
	Completed       bool      `json:"cmpleted"`
	Thumbnail       string    `json:"thumbnail"`
	Updated         time.Time `json:"updated_at"`
}

// MangaSingle contains a manga with additional fields. This struct is
// returned by requests for a single manga.
type MangaSingle struct {
	Manga
	Description string     `json:"description"`
	Chapters    []Chapter  `json:"chapters"`
	Categories  []Category `json:"rich_categories"`
	Cover       string     `json:"cover"`
	Artworks    []string   `json:"artworks"`
	Aliases     []string   `json:"alias"`
}

// Chapter of a manga.
type Chapter struct {
	ID   string `json:"oid"`
	Name string `json:"name"`
	// Updated string `json:"updatedAt"`

	// Fields only available if requested as a single object
	Order int `json:"order"`

	// Fields available if requested as chapter
	Pages []string `json:"pages"`
}

// Category of a manga.
type Category struct {
	ID   string `json:"oid"`
	Name string `json:"name"`
}

// Author of a manga.
type Author struct {
	ID        string `json:"oid"`
	Name      string `json:"name"`
	Thumbnail string `json:"thumbnail"`

	// Only available if requested through a manga
	Role string `json:"role"`
}
