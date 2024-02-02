package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"

	"github.com/gocolly/colly"
)

func main() {
	file, err := os.Create("nasa.csv")
	csvWriter := csv.NewWriter(file)
	if err != nil {
		log.Fatal("Error:", err)
	}
	defer file.Close()

	c := colly.NewCollector(
		colly.AllowedDomains("www.nasa.gov"),
		colly.MaxDepth(1),
	)

	c.OnRequest(func(r *colly.Request) {
		log.Println("Visiting", r.URL)
	})
	
	c.OnError(func(_ *colly.Response, err error) {
		log.Println("Error:", err)
	})

	c.OnHTML(".hds-gallery-items", func(e *colly.HTMLElement) {
		// Print the image link
		log.Println(e.ChildAttr("a", "href"))
		link := []string{e.ChildAttr("a", "href")}
		fmt.Println(link)

		// Write image link to CSV
		csvWriter.Write(link)

	})
	
	c.OnScraped(func(r *colly.Response) {
		log.Println("Finished", r.Request.URL)
	})

	c.Visit("https://www.nasa.gov/image-of-the-day")
}
