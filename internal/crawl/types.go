package crawl

// Request defines inputs for a crawl operation.
type Request struct {
	TargetURL string
	Depth     int
	MaxPages  int
	UserAgent string
}

// Result defines outputs for a crawl operation.
type Result struct {
	TargetURL string       `json:"target_url"`
	Pages     []PageResult `json:"pages"`
	Errors    []CrawlError `json:"errors,omitempty"`
}

// PageResult holds extracted data from a single crawled page.
type PageResult struct {
	URL             string    `json:"url"`
	StatusCode      int       `json:"status_code"`
	Title           string    `json:"title"`
	MetaDescription string    `json:"meta_description"`
	Canonical       string    `json:"canonical"`
	Headings        []Heading `json:"headings,omitempty"`
	Links           []Link    `json:"links,omitempty"`
	Images          []Image   `json:"images,omitempty"`
}

// Heading represents an HTML heading element.
type Heading struct {
	Level int    `json:"level"`
	Text  string `json:"text"`
}

// Link represents an anchor element found on a page.
type Link struct {
	Href     string `json:"href"`
	Text     string `json:"text"`
	Internal bool   `json:"internal"`
}

// Image represents an img element found on a page.
type Image struct {
	Src string `json:"src"`
	Alt string `json:"alt"`
}

// CrawlError records an error encountered while crawling a specific URL.
type CrawlError struct {
	URL     string `json:"url"`
	Message string `json:"message"`
}
