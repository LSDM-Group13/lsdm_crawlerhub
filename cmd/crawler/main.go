package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/LSDM-Group13/lsdm_crawlerhub/api"
	"golang.org/x/net/html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type PageData struct {
	pageUrl  *url.URL
	textData string
	links    []*url.URL
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

func containsScriptOrStyleAncestor(node *html.Node) bool {
	for n := node; n != nil; n = n.Parent {
		if n.Type == html.ElementNode && (n.Data == "script" || n.Data == "style") {
			return true
		}
	}
	return false
}

func (c *Crawler) insertDomain(domain string) {
	c.domainsToCrawl = append(c.domainsToCrawl, domain)
}

func ContainsLink(links []*url.URL, newLink *url.URL) bool {
	newLinkStr := newLink.String()
	for _, link := range links {
		linkStr := link.String()
		if linkStr == newLinkStr {
			return true
		}
	}
	return false
}

func (c *Crawler) requestCrawlJobs(numDomains int) {
	hubURL, err := url.Parse(c.hubBaseUrl + api.GetCrawlJobs.URL)
	if err != nil {
		fmt.Println("error forming hubURL: ", err)
		return
	}
	query := hubURL.Query()
	query.Set(api.GetCrawlJobs.Parameters.NumDomains, strconv.Itoa(numDomains))
	hubURL.RawQuery = query.Encode()

	resp, err := http.Get(hubURL.String())
	if err != nil {
		fmt.Println("error making request: ", err)
	}

	var crawlJobs api.CrawlJobs
	err = json.NewDecoder(resp.Body).Decode(&crawlJobs)
	if err != nil {
		fmt.Println("error decoding response: ", err)
	}
	c.domainsToCrawl = crawlJobs.Domains

	fmt.Println("domains received: ", c.domainsToCrawl)
}

func (c *Crawler) crawlNextDomain() (api.DomainData, error) {
	var domainName string
	c.domainsToCrawl, domainName = PopLast(c.domainsToCrawl)
	domainData := api.DomainData{
		DomainName: domainName,
		Pages:      map[string]string{},
		TimeStamp:  time.Now(),
	}

	link, err := url.Parse("https://" + domainName)
	if err != nil {
		fmt.Println("invalid domain name: ", domainName)
		return domainData, err
	}

	linksFound := []*url.URL{link}
	for maxFollow := 20; len(linksFound) > 0 && maxFollow > 0; maxFollow -= 1 {
		linksFound, link = PopLast(linksFound)

		time.Sleep(1 * time.Second)
		pageData, err := c.crawl(link)
		if err != nil {
			domainData.Pages[link.String()] = ""
			fmt.Println("error crawling ", link, ": ", err)
			continue
		}

		domainData.Pages[link.String()] = pageData.textData
		for _, newLink := range pageData.links {
			if _, ok := domainData.Pages[newLink.String()]; !ok && !ContainsLink(linksFound, newLink) {
				linksFound = append(linksFound, newLink)
			}
		}
	}

	domainData.RemoveBlankPages()
	c.domainsCrawled = append(c.domainsCrawled, domainData)

	return domainData, nil
}

func (c *Crawler) crawl(pageUrl *url.URL) (PageData, error) {
	fmt.Println("Crawling ", pageUrl)
	pageData := PageData{
		pageUrl:  pageUrl,
		textData: "",
		links:    []*url.URL{},
	}
	root, err := requestPageNodes(pageUrl)
	if err != nil {
		return pageData, err
	}

	leadingWhitespace := regexp.MustCompile(`^\s+`)
	iFrames := regexp.MustCompile(`<iframe[^>]*>(.*?)<\/iframe>`)

	nodeStack := []*html.Node{root}
	for len(nodeStack) > 0 {
		node := nodeStack[0]
		nodeStack = nodeStack[1:]

		switch nodeType := node.Type; nodeType {
		case html.TextNode:
			if !containsScriptOrStyleAncestor(node) {
				text := strings.ReplaceAll(node.Data, "\n", "")
				text = strings.ReplaceAll(text, "\t", "")
				text = leadingWhitespace.ReplaceAllString(text, "")
				text = iFrames.ReplaceAllString(text, "")

				if len(text) > 0 {
					pageData.textData += text + " "
				}
			}
		case html.ElementNode:
			for _, attr := range node.Attr {
				if attr.Key == "href" {
					if strings.ContainsAny(attr.Val, "?#") {
						continue
					}

					link, err := pageUrl.Parse(attr.Val)
					if err != nil {
						fmt.Println("failed to parse link: ", pageUrl.String(), " + ", attr.Val)
						continue
					}

					if link.Host == pageUrl.Host && !ContainsLink(pageData.links, link) {
						pageData.links = append(pageData.links, link)
					}
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

func requestPageNodes(url *url.URL) (*html.Node, error) {
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return nil // Always return nil to allow redirects
		},
	}

	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/58.0.3029.110 Safari/537.3")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Println("couldn't close response reader")
		}
	}(resp.Body)

	root, err := html.Parse(resp.Body)
	if err != nil {
		return nil, err
	}

	return root, nil
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
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Println("couldn't close response reader")
		}
	}(resp.Body)

	if resp.StatusCode != http.StatusOK {
		fmt.Println("Error:", resp.Status)
		return err
	}

	return nil
}

func main() {
	c := Crawler{
		hubBaseUrl:     "http://localhost:8869",
		maxDomains:     3,
		domainsToCrawl: nil,
		domainsCrawled: nil,
	}

	c.insertDomain("allstatehealth.com")
	dd, e := c.crawlNextDomain()
	if e != nil {
		fmt.Println(e)
	}
	fmt.Println(dd)

	//	for {
	//numJobs := 5
	//c.requestCrawlJobs(numJobs)
	//if len(c.domainsToCrawl) == 0 {
	//	break
	//}
	//for _ = range c.domainsToCrawl {
	//	domainData, err := c.crawlNextDomain()
	//	if err != nil {
	//		fmt.Println("Couldn't crawl: ", err)
	//	}
	//	fmt.Println("domain data size (bytes): ", domainData.TotalSize(), "\ndomain name: ", domainData.DomainName)
	//}
	//
	//for _ = range len(c.domainsCrawled) {
	//	err := c.postNextDomainData()
	//	if err != nil {
	//		fmt.Println("couldn't post domain data: ", err)
	//	}
	//}
	//	}
}
