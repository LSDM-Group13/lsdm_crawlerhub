package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/LSDM-Group13/lsdm_crawlerhub/api"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"io"
	"math/rand"
	"net/http"
	"os"
	"strconv"
)

func selectDomains(numJobs int) ([]string, error) {
	var domainsSelected []string

	dataSourceName := "root@tcp(localhost:3306)/LSDM_Group_Project"
	db, err := sql.Open("mysql", dataSourceName)
	if err != nil {
		return nil, err
	}
	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			fmt.Println("couldn't close database connection")
		}
	}(db)

	query := "SELECT DomainName FROM Host WHERE LastCrawledDate IS NULL LIMIT ?"
	rows, err := db.Query(query, numJobs)
	if err != nil {
		return nil, err
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			fmt.Println("couldn't close database connection")
		}
	}(rows)

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

	fmt.Println("Received domainData: ", domainData.DomainName)
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
	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			fmt.Println("couldn't close database connection")
		}
	}(db)
	fmt.Println("Connected to MySQL database")

	query := "SELECT WebPageID, HostID, WebPageURL, Data FROM WebPage WHERE Data LIKE ?"
	rows, err := db.Query(query, "%green%")
	if err != nil {
		fmt.Println(err)
	}
	defer func(rows *sql.Rows) {
		err := rows.Close()
		if err != nil {
			fmt.Println("couldn't close database connection")
		}
	}(rows)

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
	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			fmt.Println("couldn't close database connection")
		}
	}(db)
	fmt.Println("Connected to MySQL database")

	var hostID int
	err = db.QueryRow("SELECT HostID FROM Host WHERE DomainName = ?", domainData.DomainName).Scan(&hostID)
	if err != nil {
		fmt.Println("failed to find HostID")
		return err
	}

	for url, data := range domainData.Pages {
		_, err := db.Exec("INSERT INTO WebPage (HostID, WebPageURL, Data) VALUES (?, ?, ?)", hostID, url, data.Text)
		if err != nil {
			fmt.Println("failed to insert ", url)
			return err
		}
		for _, img := range data.Images {
			imageFileName := "hub_image_" + strconv.Itoa(rand.Intn(1000)) + "." + "jpg"
			imageFile, err := os.Create(imageFileName)
			if err != nil {
				fmt.Println("failed to create image file: ", err)
				continue
			}
			defer imageFile.Close()

			_, err = io.Copy(imageFile, bytes.NewReader(img.Data))
			if err != nil {
				fmt.Println("failed to save image to file: ", err)
				continue
			}

			fmt.Println("Image saved to:", imageFileName)
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
}
