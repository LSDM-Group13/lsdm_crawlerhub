package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/LSDM-Group13/lsdm_crawlerhub/api"
	"golang.org/x/net/html"
	"net/http"
	url2 "net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type PageData struct {
	pageUrl  string
	textData *string
	links    []string
}

type Crawler struct {
	hubBaseUrl     string
	maxDomains     int
	domainsToCrawl []string
	domainsCrawled []api.DomainData
}

func PopLast[T any](s []T) ([]T, T) {
	lastIdx := len(s) - 1
	last := s[lastIdx]
	newSlice := s[:lastIdx]
	return newSlice, last
}

func isValidLink(l string) bool {
	return !(strings.ContainsAny(l, ".:#?") ||
		strings.Contains(l, "wp-content"))
}

func containsScriptOrStyleAncestor(node *html.Node) bool {
	for n := node; n != nil; n = n.Parent {
		if n.Type == html.ElementNode && (n.Data == "script" || n.Data == "style") {
			return true
		}
	}
	return false
}

func requestPageNodes(url string) (*html.Node, error) {
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return nil // Always return nil to allow redirects
		},
	}
	resp, err := client.Get(url)
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

func (c *Crawler) requestCrawlJobs(numDomains int) {
	url, err := url2.Parse(c.hubBaseUrl + api.GetCrawlJobs.URL)
	if err != nil {
		fmt.Println("error forming url: ", err)
		return
	}
	query := url.Query()
	query.Set(api.GetCrawlJobs.Parameters.NumDomains, strconv.Itoa(numDomains))
	url.RawQuery = query.Encode()

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

	fmt.Println("domains received: ", c.domainsToCrawl)
}

func (c *Crawler) postNextDomainData() error {
	var domainData api.DomainData
	c.domainsCrawled, domainData = PopLast(c.domainsCrawled)

	jsonData, err := json.Marshal(domainData)
	if err != nil {
		fmt.Println("Error marshaling JSON:", err)
		return err
	}

	req, err := http.NewRequest("POST", c.hubBaseUrl+api.PostCrawlData.URL, bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Println("Error creating HTTP request:", err)
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error sending HTTP request:", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Println("Error:", resp.Status)
		return err
	}

	return nil
}

func (c *Crawler) crawl(pageUrl string) (PageData, error) {
	fmt.Println("Crawling ", pageUrl)
	pageData := PageData{
		pageUrl:  pageUrl,
		textData: new(string),
		links:    []string{},
	}
	root, err := requestPageNodes(pageUrl)
	if err != nil {
		return pageData, err
	}

	leadingWhitespace := regexp.MustCompile(`^\s+`)

	nodeStack := []*html.Node{root}
	for len(nodeStack) > 0 {
		node := nodeStack[0]
		nodeStack = nodeStack[1:]

		switch nodeType := node.Type; nodeType {
		case html.TextNode:
			if !containsScriptOrStyleAncestor(node) {
				text := strings.ReplaceAll(node.Data, "\n", "")
				text = strings.ReplaceAll(text, "\t", "")
				text = leadingWhitespace.ReplaceAllString(node.Data, "")
				if len(text) > 0 {
					*pageData.textData += text + " "
				}
			}
		case html.ElementNode:
			for _, attr := range node.Attr {
				if attr.Key == "href" {
					if !isValidLink(attr.Val) {
						continue
					}

					var link string
					if len(attr.Val) > 0 && attr.Val[0] == '/' {
						parsedUrl, _ := url2.Parse(pageUrl)
						link = parsedUrl.Scheme + "://" + parsedUrl.Host + attr.Val
					} else {
						link = pageUrl + attr.Val
					}
					pageData.links = append(pageData.links, link)
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

	return pageData, nil
}

func (c *Crawler) crawlNextDomain() (domainData api.DomainData, err error) {
	var domainName string
	c.domainsToCrawl, domainName = PopLast(c.domainsToCrawl)
	domainData = api.DomainData{
		DomainName: domainName,
		Pages:      map[string]*string{},
		TimeStamp:  time.Now(),
	}

	maxFollow := 20
	link := "http://" + domainName
	linksFound := []string{link}
	for len(linksFound) > 0 && maxFollow > 0 {
		linksFound, link = PopLast(linksFound)
		if domainData.Pages[link] != nil {
			continue
		}

		if !strings.HasSuffix(link, "/") {
			link += "/"
		}

		pageData, err := c.crawl(link)
		maxFollow -= 1
		if err != nil {
			fmt.Println("error crawling ", link, ": ", err)
			continue
		}

		domainData.Pages[link] = pageData.textData
		for _, newLink := range pageData.links {
			if domainData.Pages[newLink] == nil {
				linksFound = append(linksFound, newLink)
			}
		}
	}
	c.domainsCrawled = append(c.domainsCrawled, domainData)

	return domainData, err
}

func (c *Crawler) insertDomain(domain string) {
	c.domainsToCrawl = append(c.domainsToCrawl, domain)
}

func main() {
	c := Crawler{
		hubBaseUrl:     "http://localhost:8869",
		maxDomains:     3,
		domainsToCrawl: nil,
		domainsCrawled: nil,
	}
	//c.insertDomain("azaviculture.org")
	for {
		numJobs := 5
		c.requestCrawlJobs(numJobs)
		if len(c.domainsToCrawl) == 0 {
			break
		}
		for _ = range c.domainsToCrawl {
			domainData, err := c.crawlNextDomain()
			if err != nil {
				fmt.Println("Couldn't crawl: ", err)
			}
			fmt.Println("domain data size (bytes): ", domainData.TotalSize(), "\ndomain name: ", domainData.DomainName)
		}

		for _ = range len(c.domainsCrawled) {
			err := c.postNextDomainData()
			if err != nil {
				fmt.Println("couldn't post domain data: ", err)
			}
		}
	}
}
