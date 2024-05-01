package main

import (
	"encoding/json"
	"fmt"
	"github.com/LSDM-Group13/lsdm_crawlerhub/api"
	"github.com/LSDM-Group13/lsdm_crawlerhub/internal/hub"
	"github.com/gin-gonic/gin"
	"io"
	"math/rand/v2"
	"net/http"
	"strconv"
)

var domains = []string{"fakedomain1", "fakedomain2", "fakedomain3", "fakedomain4", "fakedomain5"}

func selectDomains(numJobs int) []string {
	var domainsSelected []string
	for range numJobs {
		domainsSelected = append(domainsSelected, domains[rand.IntN(len(domains))])
	}
	return domainsSelected
}

func getCrawlJobs(c *gin.Context) {
	domainsRequested, exists := c.GetQuery("num_domains")
	if !exists {
		domainsRequested = "1"
	}
	numDomains, err := strconv.Atoi(domainsRequested)
	if err != nil {
		numDomains = 1
	}
	jobs := api.CrawlJobs{Domains: selectDomains(numDomains)}
	c.IndentedJSON(http.StatusOK, jobs)
}

func postCrawlData(c *gin.Context) {
	requestBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "failed to read request body"})
		return
	}

	var domainData api.DomainData
	if err := json.Unmarshal(requestBody, &domainData); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	fmt.Println("Received domainData:", domainData)
}

func main() {
	hub.HelloHub()

	router := gin.Default()
	router.GET("/getCrawlJobs", getCrawlJobs)
	router.POST("/postCrawlData", postCrawlData)
	err := router.Run("localhost:8869")
	if err != nil {
		fmt.Println("error running router: ", err)
	}
}
