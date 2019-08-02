package mangarock

import (
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
)

const (
	// APIURL is the mangarock api url
	APIURL = "https://api.mangarockhd.com/query/web401"

	// APIMETAURL is mangarock's meta url
	APIMETAURL = "https://api.mangarockhd.com/meta"
)

// Client is the mangarock client
type Client struct {
	base    string
	client  *http.Client
	options map[string]string
}

// Response is the response of the mangarock web api
type Response struct {
	Code int             `json:"code"`
	Data json.RawMessage `json:"data"`
}

// WithOptions allow creating client with additional options. IE: country
func WithOptions(options map[string]string) func(*Client) {
	return func(mangarock *Client) {
		mangarock.options = options
	}
}

// New returns a brand new mangarock client
func New(options ...func(*Client)) *Client {
	mangarockClient := &Client{
		base:   APIURL,
		client: &http.Client{},
	}

	for _, option := range options {
		option(mangarockClient)
	}

	return mangarockClient
}

// Latest returns the latest mangas. It only uses the manga IDs and requests a
// list like the one that would be returned by a search. Fields like recently
// added chapters are missing, but authors are added.
func (c *Client) Latest(page int) ([]Manga, error) {
	res, err := c.get(c.base+"/mrs_latest", nil)
	if err != nil {
		return nil, err
	}
	var mangas []Manga
	if err := json.Unmarshal(res, &mangas); err != nil {
		return nil, errors.Wrap(err, "Could not unmarshal latest mangas")
	}
	ids := make([]string, len(mangas))
	for i, manga := range mangas {
		ids[i] = manga.ID
	}
	mangas, err = c.mangasByIDs(ids)
	if err != nil {
		return nil, errors.Wrap(err, "Could not get latest mangas by ids")
	}
	return c.addAuthors(mangas)
}

// Manga returns a single manga. It may contain more fields than a regular one.
func (c *Client) Manga(id string) (SingleManga, error) {
	res, err := c.post(c.base+"/info?oid="+id, nil)
	if err != nil {
		return SingleManga{}, err
	}
	var manga SingleManga
	if err := json.Unmarshal(res, &manga); err != nil {
		return SingleManga{}, errors.Wrap(err, "Could not unmarshal manga")
	}
	manga.Author = manga.Authors[0]
	return manga, nil
}

// Mangas returns a slice of mangas.
func (c *Client) Mangas(ids []string) ([]Manga, error) {
	mangas, err := c.mangasByIDs(ids)
	if err != nil {
		return nil, errors.Wrap(err, "Could not get authors mangas")
	}
	return mangas, nil
}

// Chapter returns a chapter containing its images.
func (c *Client) Chapter(id, cid string) (Chapter, error) {
	manga, err := c.Manga(id)
	if err != nil {
		return Chapter{}, errors.Wrap(err, "Could not get manga")
	}

	res, err := c.post(c.base+"/pages?oid="+cid, nil)
	if err != nil {
		return Chapter{}, errors.Wrap(err, "Could not get pages")
	}
	var pages []string
	if err := json.Unmarshal(res, &pages); err != nil {
		return Chapter{}, errors.Wrap(err, "Could not unmarhal pages")
	}

	for _, chapter := range manga.Chapters {
		if chapter.ID != cid {
			continue
		}
		chapter.Pages = pages
		return chapter, nil
	}
	return Chapter{}, errors.New("Chapter not found")
}

// Author returns an author and their mangas.
func (c *Client) Author(id string) (Author, []Manga, error) {
	authors, err := c.authorsByIDs([]string{id})
	if err != nil {
		return Author{}, nil, errors.Wrap(err,
			"Could not get authors meta data")
	}
	if len(authors) == 0 {
		return Author{}, nil,
			errors.Errorf("Author with id %s not found", id)
	}

	res, err := c.get(c.base+"/mrs_serie_related_author",
		url.Values{"oid": []string{id}})
	if err != nil {
		return Author{}, nil, errors.Wrap(err,
			"Could not get authors mangas")
	}
	var mangaIDStructs []struct {
		ID string `json:"oid"`
	}
	if err := json.Unmarshal(res, &mangaIDStructs); err != nil {
		return Author{}, nil, errors.Wrap(err,
			"Could not unmarshal authors meta data")
	}
	var mangaIDs []string
	for _, manga := range mangaIDStructs {
		mangaIDs = append(mangaIDs, manga.ID)
	}
	mangas, err := c.mangasByIDs(mangaIDs)
	if err != nil {
		return Author{}, nil, errors.Wrap(err,
			"Could not get authors mangas")
	}
	for i := range mangas {
		mangas[i].Author = authors[0]
	}
	return authors[0], mangas, nil
}

// Search searches the given query
func (c *Client) Search(query string) ([]string, error) {
	// post request body
	body := map[string]string{"type": "series", "keywords": query}
	var ids []string
	res, err := c.post(APIURL+"/mrs_search", body)
	if err := json.Unmarshal(res, &ids); err != nil {
		return nil, errors.Wrap(err, "Could not unmarshal searched mangas")
	}

	if err != nil {
		errors.Wrap(err, "Error!")
	}
	return ids, nil
}
