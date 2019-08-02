package mangarock

import (
	"encoding/json"
	"image"
	"image/png"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// SaveChapter saves given chapter slice
func SaveChapter(chapter *Chapter, path string) error {
	// log.Printf("Ch name:%s, \n Ch id:%s , \n Ch order: %d , \n Pages: %v", m.Name, m.ID, m.Order, m.Pages)
	path = path + "/"
	for index, page := range chapter.Pages {
		// log.Printf("%s", page)
		e := saveMRI(index, page, path)
		if e != nil {
			return e
		}
	}
	return nil
}

// createMangaDir creates the file with given path
func createMangaDir(path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.Mkdir(path, os.ModePerm)
	}
}

//ConvertMRItoPNG converts .mri files to .png files
func ConvertMRItoPNG(path string) error {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		log.Fatal(err)
		return err
	}

	start := time.Now()
	for _, file := range files {
		// fmt.Println(file.Name())
		r, err := os.Open(path + file.Name())
		if err != nil {
			log.Fatal("Cannot open the file")
			return err
		}
		img, _, err := image.Decode(r)
		if err != nil {
			log.Fatal("Cannot decode the MRI file")
			return err
		}

		out := strings.TrimSuffix(path+file.Name(), ".mri") + ".png"

		w, err := os.Create(out)
		if err != nil {
			log.Fatal("cannot create the outpu")
			return err
		}
		if err := png.Encode(w, img); err != nil {
			log.Fatal("Could not encode png")
			return err
		}

	}
	elapsed := time.Since(start)
	log.Printf("Elapsed time for converting: %s", elapsed)
	return nil
}

// SaveMRI saves .mri file onto disk
func saveMRI(index int, url string, path string) error {
	response, e := http.Get(url)
	if e != nil {
		log.Fatal(e)
	}
	defer response.Body.Close()
	filename := NormalizeOneDigitNumber(index) + "-" +
		lastString(strings.Split(url, "/"))

	createMangaDir(path)
	//open a file for writing
	file, err := os.Create(path + filename)
	if err != nil {
		log.Fatal(err)
		return err
	}
	defer file.Close()

	// Use io.Copy to just dump the response body to the file. This supports huge files
	_, err = io.Copy(file, response.Body)
	if err != nil {
		log.Fatal(err)
		return err
	}
	log.Println("MRI files saved successfuly!")
	return nil
}

// lastString returns last element of the splitted strin
func lastString(ss []string) string {
	return ss[len(ss)-1]
}

// NormalizeOneDigitNumber inserts '0' in front of one digit numbers [0-9].
func NormalizeOneDigitNumber(order int) string {
	if order < 10 {
		return "0" + strconv.Itoa(order)
	}
	return strconv.Itoa(order)
}

// post sends http post request
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

	var resp Response
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		return nil, errors.Wrap(err, "Could not decode response.")
	}
	if resp.Code != 0 {
		return nil, errors.Errorf("Response code %d", resp.Code)
	}

	return resp.Data, nil
}

// get sends http get request
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

	var resp Response
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		return nil, errors.Wrap(err, "Could not decode response")
	}
	if resp.Code != 0 {
		return nil, errors.Errorf("Response code %d", resp.Code)
	}

	return resp.Data, nil
}

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
