package api

type DomainData struct {
	DomainName string
	Pages      map[string]*string
}

func (dd DomainData) TotalSize() int {
	totalSize := 0
	for _, p := range dd.Pages {
		totalSize += len(*p)
	}

	return totalSize
}

type CrawlJobs struct {
	Domains []string `json:"domains"`
}

type APIEndpoint struct {
	URL        string
	Parameters struct {
		NumDomains string
	}
}

var (
	GetCrawlJobs = APIEndpoint{
		URL: "/getCrawlJobs",
		Parameters: struct{ NumDomains string }{
			NumDomains: "num_domains",
		},
	}

	PostCrawlData = APIEndpoint{
		URL: "/postCrawlData",
		Parameters: struct{ NumDomains string }{
			NumDomains: "num_domains",
		},
	}
)
