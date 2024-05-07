package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/LSDM-Group13/lsdm_crawlerhub/api"
	"golang.org/x/net/html"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type PageData struct {
	pageUrl  *url.URL
	textData string
	links    []*url.URL
	images   []api.Image
}

func (pd *PageData) updateText(s string) {
	if len(s) > 0 {
		pd.textData += " " + s
	}
}

type Crawler struct {
	hubBaseUrl       string
	maxDomains       int
	domainsToCrawl   []string
	domainsCrawled   []api.DomainData
	maxImagesPerPage int
	maxLinkDepth     int
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
		if link.String() == newLinkStr {
			return true
		}
	}
	return false
}

func (c *Crawler) requestCrawlJobs(numDomains int) error {
	hubURL, err := url.Parse(c.hubBaseUrl + api.GetCrawlJobs.URL)
	if err != nil {
		fmt.Println("error forming hubURL: ", err)
		return err
	}
	query := hubURL.Query()
	query.Set(api.GetCrawlJobs.Parameters.NumDomains, strconv.Itoa(numDomains))
	hubURL.RawQuery = query.Encode()

	resp, err := http.Get(hubURL.String())
	if err != nil {
		fmt.Println("error making request: ", err)
		return err
	}

	var crawlJobs api.CrawlJobs
	err = json.NewDecoder(resp.Body).Decode(&crawlJobs)
	if err != nil {
		fmt.Println("error decoding response: ", err)
		return err
	}
	c.domainsToCrawl = crawlJobs.Domains

	fmt.Println("domains received: ", c.domainsToCrawl)
	return nil
}

func (c *Crawler) crawlLinks(link *url.URL) map[string]api.PageContent {
	linksFound := []*url.URL{link}
	pageContentMap := make(map[string]api.PageContent)
	for linksFollowed := 0; len(linksFound) > 0 && linksFollowed < c.maxLinkDepth; linksFollowed += 1 {
		linksFound, link = PopLast(linksFound)

		time.Sleep(1 * time.Second)
		pageData, err := c.crawl(link)
		if err != nil {
			fmt.Println("error crawling ", link, ": ", err)
		}

		pageContentMap[link.String()] = api.PageContent{
			Text:   pageData.textData,
			Images: pageData.images,
		}

		for _, newLink := range pageData.links {
			if _, ok := pageContentMap[newLink.String()]; !ok && !ContainsLink(linksFound, newLink) {
				linksFound = append(linksFound, newLink)
			}
		}
	}

	return pageContentMap
}

func (c *Crawler) crawlNextDomain() (api.DomainData, error) {
	var domainName string
	c.domainsToCrawl, domainName = PopLast(c.domainsToCrawl)
	domainData := api.DomainData{
		DomainName: domainName,
		Pages:      make(map[string]api.PageContent),
		TimeStamp:  time.Now(),
	}

	link, err := url.Parse("https://" + domainName + "/")
	if err != nil {
		fmt.Println("invalid domain name: ", domainName)
		return domainData, err
	}

	domainData.Pages = c.crawlLinks(link)
	domainData.RemoveBlankPages()
	c.domainsCrawled = append(c.domainsCrawled, domainData)

	return domainData, nil
}

func isValidLink(l string) bool {
	return !strings.ContainsAny(l, "?#") && !strings.Contains(l, "wp-content") && !strings.HasSuffix(l, ".css") && !strings.HasSuffix(l, ".torrent")
}

func extractText(node *html.Node) string {
	if containsScriptOrStyleAncestor(node) {
		return ""
	}

	leadingWhitespace := regexp.MustCompile(`^\s+`)
	iFrames := regexp.MustCompile(`<iframe[^>]*>(.*?)<\/iframe>`)

	text := strings.ReplaceAll(node.Data, "\n", "")
	text = strings.ReplaceAll(text, "\t", "")
	text = leadingWhitespace.ReplaceAllString(text, "")
	text = iFrames.ReplaceAllString(text, "")

	return text
}

func extractLinkText(node *html.Node) string {
	hrefAttr, attrFound := findAttrByKey(node.Attr, "href")
	if !attrFound || !isValidLink(hrefAttr.Val) {
		return ""
	}

	return hrefAttr.Val
}

func findAttrByKey(attributes []html.Attribute, key string) (html.Attribute, bool) {
	for _, attr := range attributes {
		if attr.Key == key {
			return attr, true
		}
	}

	return html.Attribute{}, false
}

func (pd *PageData) updateLinks(linkText string) {
	link, err := pd.pageUrl.Parse(linkText)
	if err != nil {
		fmt.Println("failed to parse link: ", pd.pageUrl.String(), " + ", linkText)
	}

	if link.Host == pd.pageUrl.Host && !ContainsLink(pd.links, link) {
		pd.links = append(pd.links, link)
	}
}

func (c *Crawler) crawl(pageUrl *url.URL) (PageData, error) {
	fmt.Println("Crawling ", pageUrl)
	pageData := PageData{
		pageUrl:  pageUrl,
		textData: "",
		links:    []*url.URL{},
		images:   make([]api.Image, 0, c.maxImagesPerPage),
	}
	root, err := requestPageNodes(pageUrl)
	if err != nil {
		return pageData, err
	}

	imagesFound := 0
	nodeStack := []*html.Node{root}
	var node *html.Node
	for len(nodeStack) > 0 {
		nodeStack, node = PopLast(nodeStack)
		for sib := node.FirstChild; sib != nil; sib = sib.NextSibling {
			nodeStack = append(nodeStack, sib)
		}

		if node.Type == html.TextNode {
			pageData.updateText(extractText(node))
		} else if node.Type != html.ElementNode {
			continue
		}

		if node.Data == "a" {
			pageData.updateLinks(extractLinkText(node))
		} else if node.Data == "img" && imagesFound < c.maxImagesPerPage {
			srcAttr, attrFound := findAttrByKey(node.Attr, "src")
			if !attrFound {
				continue
			}

			imageUrl, err := pageUrl.Parse(srcAttr.Val)
			if err != nil {
				fmt.Println("failed to parse image URL: ", srcAttr.Val)
				continue
			}

			resp, err := http.Get(imageUrl.String())
			if err != nil {
				fmt.Println("failed to download image: ", err)
				continue
			}
			defer resp.Body.Close()
			imagesFound += 1

			parts := strings.Split(imageUrl.String(), ".")
			ext := parts[len(parts)-1]
			imageFileName := "image_" + strconv.Itoa(rand.Intn(1000)) + "." + ext
			imageFile, err := os.Create(imageFileName)
			if err != nil {
				fmt.Println("failed to create image file: ", err)
				continue
			}
			defer imageFile.Close()

			imgBytes, _ := io.ReadAll(resp.Body)
			image := api.Image{Name: imageFileName, Data: imgBytes}
			pageData.images = append(pageData.images, image)

			_, err = io.Copy(imageFile, resp.Body)
			if err != nil {
				fmt.Println("failed to save image to file: ", err)
				continue
			}

			fmt.Println("Image downloaded and saved to:", imageFileName)
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
		hubBaseUrl:       "http://localhost:8869",
		maxDomains:       3,
		domainsToCrawl:   nil,
		domainsCrawled:   nil,
		maxImagesPerPage: 2,
		maxLinkDepth:     20,
	}

	//c.insertDomain("allstatehealth.com")
	//dd, e := c.crawlNextDomain()
	//if e != nil {
	//	fmt.Println(e)
	//}
	//fmt.Println(dd)

	numJobs := 1
	c.requestCrawlJobs(numJobs)
	for range c.domainsToCrawl {
		domainData, err := c.crawlNextDomain()
		if err != nil {
			fmt.Println("Couldn't crawl: ", err)
		}
		fmt.Println("domain data size (bytes): ", domainData.TotalSize(), "\ndomain name: ", domainData.DomainName)
	}

	for range len(c.domainsCrawled) {
		err := c.postNextDomainData()
		if err != nil {
			fmt.Println("couldn't post domain data: ", err)
		}
	}
}
