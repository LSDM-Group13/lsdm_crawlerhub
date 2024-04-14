package main

import (
	"encoding/json"
	"fmt"
	"github.com/LSDM-Group13/lsdm_crawlerhub/api"
	"github.com/LSDM-Group13/lsdm_crawlerhub/internal/crawler"
	"net/http"
	url2 "net/url"
	"strconv"
)

type Crawler struct {
	hubBaseUrl     string
	maxDomains     int
	domainsToCrawl []string
	domainsCrawled []string
}

func (c *Crawler) requestCrawlJobs(numDomains int) {
	url, err := url2.Parse(c.hubBaseUrl + api.GetCrawlJobs.URL)
	if err != nil {
		fmt.Println("error forming url: ", err)
		return
	}
	query := url.Query()
	query.Set(api.GetCrawlJobs.Parameters.NumDomains, strconv.Itoa(numDomains))
	url.RawQuery = query.Encode()
	fmt.Println(url.String())

	resp, err := http.Get(url.String())
	if err != nil {
		fmt.Println("error make request: ", err)
	}

	var crawlJobs api.CrawlJobs
	err = json.NewDecoder(resp.Body).Decode(&crawlJobs)
	if err != nil {
		fmt.Println("error decoding response: ", err)
	}

	c.domainsToCrawl = crawlJobs.Domains
	fmt.Println(c.domainsToCrawl)
}

func main() {
	crawler.HelloCrawler()
	c := Crawler{
		hubBaseUrl:     "http://localhost:8869",
		maxDomains:     3,
		domainsToCrawl: nil,
		domainsCrawled: nil,
	}

	c.requestCrawlJobs(2)
}
