package api

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
)
