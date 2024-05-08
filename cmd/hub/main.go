package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/LSDM-Group13/lsdm_crawlerhub/api"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gocql/gocql"
	"io"
	"net/http"
	"strconv"
)

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

	cluster := gocql.NewCluster("localhost:9042")
	cluster.Keyspace = "lsdm_images"
	session, err := cluster.CreateSession()
	defer session.Close()
	for url, data := range domainData.Pages {
		_, err := db.Exec("INSERT INTO WebPage (HostID, WebPageURL, Data) VALUES (?, ?, ?)", hostID, url, data.Text)
		if err != nil {
			fmt.Println("failed to insert ", url)
			return err
		}

		var webPageID int
		err = db.QueryRow("SELECT WebPageID FROM WebPage WHERE HostID = ?", hostID).Scan(&webPageID)
		if err != nil {
			fmt.Println("failed to find HostID")
			return err
		}
		for _, img := range data.Images {
			imgID := gocql.TimeUUID()
			if err != nil {
				fmt.Println("couldn't create cql session: ", err)
				return err
			}
			q := session.Query("INSERT INTO lsdm_images.images (image_id, image_data, webpage_id) VALUES (?, ?, ?)",
				imgID, img.Data, webPageID)

			if err := q.Exec(); err != nil {
				fmt.Println("couldn't execute query: ", err)
				return err
			}
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
