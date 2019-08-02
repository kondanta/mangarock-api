package mangarock

import (
	"encoding/json"
	"io"
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

// response is the response of the mangarock web api
type response struct {
	Code int             `json:"code"`
	Data json.RawMessage `json:"data"`
}

// WithOptions ...
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

func (c *Client) post(url string, body interface{}) (json.RawMessage, error) {
	pr, pw := io.Pipe()
	go func() {
		defer pw.Close()
		json.NewEncoder(pw).Encode(body)
	}()

	req, err := http.NewRequest(http.MethodPost, url, pr)
	if err != nil {
		return nil, errors.Wrap(err, "Could not create Post request.")
	}

	res, err := c.client.Do(req)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not post to %v .", url)
	}
	defer res.Body.Close()

	var resp response
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		return nil, errors.Wrap(err, "Could not decode response.")
	}
	if resp.Code != 0 {
		return nil, errors.Errorf("Response code %d", resp.Code)
	}

	return resp.Data, nil
}

func (c *Client) get(url string, query url.Values) (json.RawMessage, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "Could not create the GET request.")
	}

	req.URL.RawQuery = query.Encode()
	res, err := c.client.Do(req)
	if err != nil {
		return nil, errors.Wrapf(err, "Could not GET %v", url)
	}
	defer res.Body.Close()

	var resp response
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		return nil, errors.Wrap(err, "Could not decode response")
	}
	if resp.Code != 0 {
		return nil, errors.Errorf("Response code %d", resp.Code)
	}

	return resp.Data, nil
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
func (c *Client) Manga(id string) (MangaSingle, error) {
	res, err := c.post(c.base+"/info?oid="+id, nil)
	if err != nil {
		return MangaSingle{}, err
	}
	var manga MangaSingle
	if err := json.Unmarshal(res, &manga); err != nil {
		return MangaSingle{}, errors.Wrap(err, "Could not unmarshal manga")
	}
	manga.Author = manga.Authors[0]
	return manga, nil
}

// NOTE: From below was taken from bake's mangarock api.
// I'll change the implementation, so instead of using it as a dependency
// I chose re-inventing the wheel.

// mangasByIDs returns a list of mangas based on IDs. Can be used to unify
// manga results that are slightly different.
func (c *Client) mangasByIDs(ids []string) ([]Manga, error) {
	res, err := c.post(APIMETAURL, ids)
	if err != nil {
		return nil, errors.Wrap(err, "Could not get meta data by manga ids")
	}
	var mangaMap map[string]Manga
	if err := json.Unmarshal(res, &mangaMap); err != nil {
		return nil, errors.Wrap(err, "Could not unmarshal mangas by ids")
	}
	var mangas []Manga
	for _, id := range ids {
		if manga, ok := mangaMap[id]; ok {
			mangas = append(mangas, manga)
		}
	}
	return mangas, nil
}

// addAuthors adds authors to mangas based on their IDs.
func (c *Client) addAuthors(mangas []Manga) ([]Manga, error) {
	var ids []string
	for _, manga := range mangas {
		ids = append(ids, manga.AuthorIDs...)
	}
	authors, err := c.authorsByIDs(ids)
	if err != nil {
		return nil, errors.Wrap(err, "Could not get authors by ids")
	}
	authorMap := map[string]Author{}
	for _, author := range authors {
		authorMap[author.ID] = author
	}

	for i, manga := range mangas {
		for _, id := range manga.AuthorIDs {
			mangas[i].Authors = append(mangas[i].Authors, authorMap[id])
		}
		if len(mangas[i].Authors) == 0 {
			continue
		}
		mangas[i].Author = mangas[i].Authors[0]
	}

	return mangas, nil
}

// authorsByIDs returns a slice of authors by their IDs.
func (c *Client) authorsByIDs(ids []string) ([]Author, error) {
	res, err := c.post(APIMETAURL, ids)
	if err != nil {
		return nil, errors.Wrap(err, "Could not get meta data by author ids")
	}
	var authorMap map[string]Author
	if err := json.Unmarshal(res, &authorMap); err != nil {
		return nil, errors.Wrap(err, "Could not unmarshal authors by ids")
	}
	var authors []Author
	for _, author := range authorMap {
		authors = append(authors, author)
	}
	return authors, nil
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
