package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/LSDM-Group13/lsdm_crawlerhub/api"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
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
	err = postDomainDataToDB(domainData)
	if err != nil {
		fmt.Println(err)
	}
}

func printDBTestData() {
	dataSourceName := "root@tcp(localhost:3306)/LSDM_Group_Project"
	db, err := sql.Open("mysql", dataSourceName)
	if err != nil {
		fmt.Println("couldn't open")
		fmt.Println(err)
	}
	defer db.Close()
	fmt.Println("Connected to MySQL database")

	query := "SELECT WebPageID, HostID, WebPageURL, Data FROM WebPage WHERE Data LIKE ?"
	rows, err := db.Query(query, "%green%")
	if err != nil {
		fmt.Println(err)
	}
	defer rows.Close()

	for rows.Next() {
		var webpageID, hostID int
		var webpageURL, data string
		if err := rows.Scan(&webpageID, &hostID, &webpageURL, &data); err != nil {
			fmt.Println(err)
		}
		fmt.Printf("%d, %d, %s, %s\n", webpageID, hostID, webpageURL, data)
	}
	if err := rows.Err(); err != nil {
		fmt.Println(err)
	}

	fmt.Println("Search completed successfully")
}

func postDomainDataToDB(domainData api.DomainData) error {
	dataSourceName := "root@tcp(localhost:3306)/LSDM_Group_Project"
	db, err := sql.Open("mysql", dataSourceName)
	if err != nil {
		fmt.Println("couldn't open")
		fmt.Println(err)
		return err
	}
	defer db.Close()
	fmt.Println("Connected to MySQL database")

	var hostID int
	err = db.QueryRow("SELECT HostID FROM Host WHERE DomainName = ?", domainData.DomainName).Scan(&hostID)
	if err != nil {
		fmt.Println("failed to find HostID")
		return err
	}

	for url, data := range domainData.Pages {
		_, err := db.Exec("INSERT INTO WebPage (HostID, WebPageURL, Data) VALUES (?, ?, ?)", hostID, url, data)
		if err != nil {
			fmt.Println("failed to insert ", url)
			return err
		}
	}

	_, err = db.Exec("UPDATE Host SET LastCrawledDate = ? WHERE HostID = ?", domainData.TimeStamp, hostID)
	if err != nil {
		fmt.Println("failed to update LastCrawledDate")
		return err
	}

	fmt.Println("Page data inserted into WebPage table successfully")
	return nil
}

func main() {
	router := gin.Default()
	router.GET("/getCrawlJobs", getCrawlJobs)
	router.POST("/postCrawlData", postCrawlData)
	err := router.Run("localhost:8869")
	if err != nil {
		fmt.Println("error running router: ", err)
	}

	if err != nil {
		println(err)
	}
}
