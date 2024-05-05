package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/LSDM-Group13/lsdm_crawlerhub/api"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"io"
	"net/http"
	"strconv"
)

func selectDomains(numJobs int) ([]string, error) {
	var domainsSelected []string

	dataSourceName := "root@tcp(localhost:3306)/LSDM_Group_Project"
	db, err := sql.Open("mysql", dataSourceName)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	query := "SELECT DomainName FROM Host WHERE LastCrawledDate IS NULL LIMIT ?"
	rows, err := db.Query(query, numJobs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var domainName string
		if err := rows.Scan(&domainName); err != nil {
			return nil, err
		}
		domainsSelected = append(domainsSelected, domainName)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(domainsSelected) == 0 {
		return domainsSelected, fmt.Errorf("no more domains to crawl")
	}

	return domainsSelected, nil
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
	newDomains, err := selectDomains(numDomains)
	if err != nil {
		fmt.Println("couldn't get jobs from db")
		return
	}
	jobs := api.CrawlJobs{Domains: newDomains}
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

	domains, err := selectDomains(5)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(domains)
	}
}
