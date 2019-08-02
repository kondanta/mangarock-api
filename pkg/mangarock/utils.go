package mangarock

import (
	"image"
	"image/png"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
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
