package main

import (
	"encoding/csv"
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
	GALLERY_URL = "https://www.nasa.gov/image-of-the-day"
	FILE = "img-of-the-day.csv"
)

type NasaImage struct {
	url string
	title string
	description string
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
	log.Println("Starting the scraping process...")
	var imgUrls []string
	
	// Scrap the links to the images in the gallery
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
		imgUrls = append(imgUrls, url)
	})
	c.Visit(GALLERY_URL) // Start the scraping
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
	var wg sync.WaitGroup
	counter := 0
	wg.Add(len(imgUrls))
	for _, imgUrl := range imgUrls {
		counter++
		go func(imgUrl string) {
			nasaImage := NasaImage{url: imgUrl}
			defer wg.Done()
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
			// ADD DOWNLOADING IMAGE HERE
			c.OnHTML(".hds-attachment-single__image", func(e *colly.HTMLElement) {
				imageUrl := e.ChildAttr("img", "src")
				// Ensure that the image URL is not empty
				if imageUrl != "" {
					// Download the image using the random user agent
					err := downloadFile(filepath.Base(imageUrl), imageUrl, randomUserAgent(userAgents))
					if err != nil {
						log.Println("Error while downloading image:", err)
					} else {
						log.Println("Image downloaded successfully:", imageUrl)
					}
				}
			})
			// Description and Title extraction
			c.OnHTML(".hds-attachment-single__content", func(e *colly.HTMLElement) {
				nasaImage.title = e.ChildText("h1")
				nasaImage.description = e.ChildText("p")
			})
			c.Visit(nasaImage.url)
			err = csvWriter.Write([]string{nasaImage.url, nasaImage.title, nasaImage.description})
		}(imgUrl)
	}
	wg.Wait()
	log.Println("Scraping completed successfully.")
}