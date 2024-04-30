package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/LSDM-Group13/lsdm_crawlerhub/api"
	"github.com/LSDM-Group13/lsdm_crawlerhub/internal/crawler"
	"golang.org/x/net/html"
	"net/http"
	url2 "net/url"
	"strconv"
)

type pageURL string
type wordCounts map[string]int

type DomainData struct {
	domainName string
	pages      map[pageURL]wordCounts
}

type Crawler struct {
	hubBaseUrl     string
	maxDomains     int
	domainsToCrawl []string
	domainsCrawled []DomainData
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

func crawlEveryNode(root *html.Node) {
	nodeStack := []*html.Node{root}
	for len(nodeStack) > 0 {
		node := nodeStack[0]
		nodeStack = nodeStack[1:]

		switch nodeType := node.Type; nodeType {
		case html.TextNode:
			//fmt.Println(node.Data)
		case html.ElementNode:
			for _, attr := range node.Attr {
				if attr.Key == "href" {
					fmt.Println(attr.Val)
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
}

func (c *Crawler) crawlNextDomain() {
	domain := c.domainsToCrawl[len(c.domainsToCrawl)-1]
	resp, err := http.Get(domain)
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(resp.Status)

	s := bufio.NewReader(resp.Body)
	root, err := html.Parse(s)
	if err != nil {
		fmt.Println(err)
	}

	//fmt.Println(root.Data)
	crawlEveryNode(root)

	c.domainsToCrawl = c.domainsToCrawl[:len(c.domainsToCrawl)-1]
	//TODO: construct real domainData bundle
	c.domainsCrawled = append(c.domainsCrawled, DomainData{
		domainName: "",
		pages:      nil,
	})
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
	c.crawlNextDomain()

}
