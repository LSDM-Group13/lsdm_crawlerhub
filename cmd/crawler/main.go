package main

import (
	"encoding/json"
	"fmt"
	"github.com/LSDM-Group13/lsdm_crawlerhub/api"
	"github.com/LSDM-Group13/lsdm_crawlerhub/internal/crawler"
	"golang.org/x/net/html"
	"net/http"
	url2 "net/url"
	"strconv"
)

type wordCounts map[string]int
type domainNames []string

type DomainData struct {
	domainName string
	pages      map[string]wordCounts
}

type Crawler struct {
	hubBaseUrl     string
	maxDomains     int
	domainsToCrawl domainNames
	domainsCrawled []DomainData
}

func (dns *domainNames) popLast() string {
	lastIdx := len(*dns) - 1
	last := (*dns)[lastIdx]
	*dns = (*dns)[:lastIdx]
	return last
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

func requestPageNodes(url string) (*html.Node, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	if resp.Status != "200 OK" {
		return nil, fmt.Errorf("not 200 OK")
	}

	defer resp.Body.Close()
	root, err := html.Parse(resp.Body)
	if err != nil {
		return nil, err
	}

	return root, nil
}

func (c *Crawler) crawl(domain string) (DomainData, error) {
	domainData := DomainData{
		domainName: domain,
		pages:      map[string]wordCounts{},
	}
	domainData.pages[domain] = wordCounts{}

	root, err := requestPageNodes(domain)
	if err != nil {
		return domainData, err
	}

	nodeStack := []*html.Node{root}
	for len(nodeStack) > 0 {
		node := nodeStack[0]
		nodeStack = nodeStack[1:]

		switch nodeType := node.Type; nodeType {
		case html.TextNode:
			//TODO: make word-count dictionary
			domainData.pages[domain]["a"] += 1
		case html.ElementNode:
			for _, attr := range node.Attr {
				if attr.Key == "href" {
					//TODO: filter out non-relative links ("https://....")
					pageUrl := fmt.Sprintf("%s%s\n", domain, attr.Val)
					domainData.pages[pageUrl] = wordCounts{}
				}
			}
		}

		child := node.FirstChild
		if child == nil {
			continue
		}

		for sib := child; sib != nil; sib = sib.NextSibling {
			nodeStack = append(nodeStack, sib)
		}
	}

	return domainData, nil
}

func (c *Crawler) crawlNextDomain() (err error) {
	domain := c.domainsToCrawl.popLast()
	domainData, err := c.crawl(domain)
	if err == nil {
		c.domainsCrawled = append(c.domainsCrawled, domainData)
	}

	return err
}

func (c *Crawler) insertDomain(domain string) {
	c.domainsToCrawl = append(c.domainsToCrawl, domain)
}

func main() {
	crawler.HelloCrawler()
	c := Crawler{
		hubBaseUrl:     "http://localhost:8869",
		maxDomains:     3,
		domainsToCrawl: nil,
		domainsCrawled: nil,
	}

	//c.requestCrawlJobs(2)
	c.insertDomain("https://allstatehealth.com")
	err := c.crawlNextDomain()
	if err != nil {
		fmt.Println("Couldn't crawl: ", err)
	}
	fmt.Println(c.domainsCrawled[0])

}
