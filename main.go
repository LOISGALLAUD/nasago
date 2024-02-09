package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/gocolly/colly"
)

//----------------------------------------------------------------------------------//

const (
	DOMAIN = "www.nasa.gov"
	MAX_PAGE = 1
	IMG_PER_PAGE = 40
	GALLERY_URL = "https://www.nasa.gov/image-of-the-day/page/"
	FILE = "img-of-the-day.csv"
)

type NasaImage struct {
	title string
	description string
	url string
}

var userAgents = []string{
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/109.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/109.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/108.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/108.0.0.0 Safari/537.36",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/108.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.1 Safari/605.1.15",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 13_1) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.1 Safari/605.1.15",
}

func randomUserAgent(list []string) string {
	randomIndex := rand.Intn(len(list))
	return list[randomIndex]
}

func scrapeGallery(page int, ch chan string) {
	c := colly.NewCollector(
		colly.AllowedDomains(DOMAIN),
		colly.MaxDepth(1),
	)
	c.OnRequest(func(r *colly.Request) {
		r.Headers.Set("User-Agent", randomUserAgent(userAgents))
	})
	c.OnError(func(_ *colly.Response, err error) {
		log.Println("Error while scraping the gallery:", err)
	})
	c.OnHTML(".hds-gallery-item-single", func(e *colly.HTMLElement) {
		// Extract url from the element
		url := e.ChildAttr("a", "href")
		// Send the url to the channel
		ch <- url
	})
	c.Visit(fmt.Sprintf("%s%d", GALLERY_URL, page))
	fmt.Printf("Scraping the page at link: %s%d\n", GALLERY_URL, page)
}

func downloadFile(filename, url, userAgent string) error {
    // Get the current working directory
    currentDir, err := os.Getwd()
    if err != nil {
        return err
    }

    // Create the full filepath by joining the current directory and the filename
    filepath := filepath.Join(currentDir, "images", filename)

    // Create the file
    out, err := os.Create(filepath)
    if err != nil {
        return err
    }
    defer out.Close()

    // Create a new HTTP client with a custom user agent
    client := &http.Client{
        CheckRedirect: func(req *http.Request, via []*http.Request) error {
            req.Header.Set("User-Agent", userAgent)
            return nil
        },
    }

    // Create a new request
    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        return err
    }

    // Perform the request
    resp, err := client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    // Write the body to the file
    _, err = io.Copy(out, resp.Body)
    if err != nil {
        return err
    }

    return nil
}

//----------------------------------------------------------------------------------//

func main() {
	// Create a folder to store the images
	err := os.Mkdir("images", 0755)
	if err != nil {
		log.Println("Error while creating the images folder:", err)
	}

	log.Println("Starting the scraping process...")
	page := 1
	ch := make(chan string, IMG_PER_PAGE*MAX_PAGE)
	var wg_gallery sync.WaitGroup
	wg_gallery.Add(MAX_PAGE)
	
	// Scrap the links to the images in the gallery
	for page <= MAX_PAGE {
		go func(page int) {
            defer wg_gallery.Done()
            scrapeGallery(page, ch)
        }(page)
        page++
		}
	wg_gallery.Wait()
	close(ch)
	log.Println("Scraping the gallery completed successfully.")
	
	//----------------------------------------------------//

	// Scrap each individual image
	log.Println("Scraping individual images...")

	file, err := os.Create(FILE)
	if err != nil {
		log.Fatal("Error:", err)
	}
	defer file.Close()
	csvWriter := csv.NewWriter(file)
	defer csvWriter.Flush()
	err = csvWriter.Write([]string{"title", "description", "url"})
	if err != nil {
		log.Fatal("Cannot write to file", err)
	}
	var wg_img sync.WaitGroup
	counter := 0
	wg_img.Add(len(ch))
	fmt.Printf("Longueur de imgUrls%d", len(ch))
	for imgUrl := range ch {
		counter++
		go func(imgUrl string) {
			nasaImage := NasaImage{url: imgUrl}
			defer wg_img.Done()
			c := colly.NewCollector(
				colly.AllowedDomains(DOMAIN),
				colly.MaxDepth(1),
			)
			c.OnRequest(func(r *colly.Request) {
				r.Headers.Set("User-Agent", randomUserAgent(userAgents))
			})
			c.OnError(func(_ *colly.Response, err error) {
				log.Println("Error while scraping an image:", err)
			})
			// Image extraction
			c.OnHTML(".hds-attachment-single__image", func(e *colly.HTMLElement) {
				imageUrl := e.ChildAttr("img", "src")
				// Ensure that the image URL is not empty
				if imageUrl != "" {
					imgName := filepath.Base(imageUrl)
					// Download the image using the random user agent
					err := downloadFile(imgName, imageUrl, randomUserAgent(userAgents))
					if err != nil {
						log.Println("Error while downloading image:", err)
					} else {
						log.Println("Image downloaded successfully:", imgName)
					}
				}
			})
			// Description and Title extraction
			c.OnHTML(".hds-attachment-single__content", func(e *colly.HTMLElement) {
				nasaImage.title = e.ChildText("h1")
				nasaImage.description = e.ChildText("p")
			})
			c.Visit(nasaImage.url)
			err = csvWriter.Write([]string{nasaImage.title, nasaImage.description, nasaImage.url})
		}(imgUrl)
	}
	wg_img.Wait()
	log.Println("Scraping completed successfully.")
}