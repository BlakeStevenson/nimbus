package main

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// NewznabClient represents a Newznab API client
type NewznabClient struct {
	BaseURL string
	APIKey  string
	Client  *http.Client
}

// NewznabResponse represents the XML response from Newznab API
type NewznabResponse struct {
	XMLName xml.Name       `xml:"rss"`
	Channel NewznabChannel `xml:"channel"`
}

// NewznabChannel represents the channel element in Newznab response
type NewznabChannel struct {
	Title       string        `xml:"title"`
	Description string        `xml:"description"`
	Items       []NewznabItem `xml:"item"`
}

// NewznabItem represents a single release/item in the feed
type NewznabItem struct {
	Title       string           `xml:"title"`
	GUID        string           `xml:"guid"`
	Link        string           `xml:"link"`
	Comments    string           `xml:"comments"`
	PubDate     string           `xml:"pubDate"`
	Category    string           `xml:"category"`
	Description string           `xml:"description"`
	Enclosure   NewznabEnclosure `xml:"enclosure"`
	Attributes  []NewznabAttr    `xml:"attr"`
}

// NewznabEnclosure represents the enclosure (download link) element
type NewznabEnclosure struct {
	URL    string `xml:"url,attr"`
	Length int64  `xml:"length,attr"`
	Type   string `xml:"type,attr"`
}

// NewznabAttr represents custom Newznab attributes
type NewznabAttr struct {
	Name  string `xml:"name,attr"`
	Value string `xml:"value,attr"`
}

// SearchParams represents search parameters for Newznab API
type SearchParams struct {
	Query      string   // Search query (q parameter)
	Categories []string // Category IDs
	TVDBID     string   // TVDB ID for TV shows
	TVRageID   string   // TVRage ID
	IMDBID     string   // IMDB ID for movies
	Season     int      // Season number (0 means not specified)
	Episode    int      // Episode number (0 means not specified)
	Limit      int      // Max results (default 100)
	Offset     int      // Offset for pagination
}

// Release represents a normalized release from Newznab
type Release struct {
	ID          string            `json:"id"`
	Title       string            `json:"title"`
	GUID        string            `json:"guid"`
	Link        string            `json:"link"`
	Comments    string            `json:"comments"`
	PublishDate time.Time         `json:"publish_date"` // Changed to match IndexerRelease
	Category    string            `json:"category"`
	Size        int64             `json:"size"`
	Description string            `json:"description"`
	DownloadURL string            `json:"download_url"` // Changed to match IndexerRelease
	Attributes  map[string]string `json:"attributes"`
	IndexerID   string            `json:"indexer_id,omitempty"`   // Added for IndexerRelease compatibility
	IndexerName string            `json:"indexer_name,omitempty"` // Added for IndexerRelease compatibility
}

// NewNewznabClient creates a new Newznab client
func NewNewznabClient(baseURL, apiKey string) *NewznabClient {
	return &NewznabClient{
		BaseURL: strings.TrimSuffix(baseURL, "/"),
		APIKey:  apiKey,
		Client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Search performs a search on the Newznab indexer
func (c *NewznabClient) Search(params SearchParams) ([]Release, error) {
	if params.Limit == 0 {
		params.Limit = 100
	}

	apiURL := fmt.Sprintf("%s/api", c.BaseURL)

	// Build query parameters
	queryParams := url.Values{}
	queryParams.Set("t", "search")
	queryParams.Set("apikey", c.APIKey)
	queryParams.Set("limit", strconv.Itoa(params.Limit))
	queryParams.Set("offset", strconv.Itoa(params.Offset))

	if params.Query != "" {
		queryParams.Set("q", params.Query)
	}

	if len(params.Categories) > 0 {
		queryParams.Set("cat", strings.Join(params.Categories, ","))
	}

	// Make the request
	resp, err := c.Client.Get(apiURL + "?" + queryParams.Encode())
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	return c.parseResponse(resp.Body)
}

// SearchTV performs a TV show search on the Newznab indexer
func (c *NewznabClient) SearchTV(params SearchParams) ([]Release, error) {
	if params.Limit == 0 {
		params.Limit = 100
	}

	apiURL := fmt.Sprintf("%s/api", c.BaseURL)

	// Build query parameters
	queryParams := url.Values{}
	queryParams.Set("t", "tvsearch")
	queryParams.Set("apikey", c.APIKey)
	queryParams.Set("limit", strconv.Itoa(params.Limit))
	queryParams.Set("offset", strconv.Itoa(params.Offset))

	// Set tvdbid if available
	if params.TVDBID != "" {
		queryParams.Set("tvdbid", params.TVDBID)
		// For season searches, also include query (e.g., "S01") to help prioritize season packs
		// This is safe because the query is just a season identifier, not a title search
		if params.Query != "" && params.Season > 0 && params.Episode == 0 {
			queryParams.Set("q", params.Query)
		}
	} else if params.Query != "" {
		// Only use query if we don't have a tvdbid
		queryParams.Set("q", params.Query)
	}

	if len(params.Categories) > 0 {
		queryParams.Set("cat", strings.Join(params.Categories, ","))
	}

	if params.TVRageID != "" {
		queryParams.Set("rid", params.TVRageID)
	}

	if params.Season > 0 {
		queryParams.Set("season", strconv.Itoa(params.Season))
	}

	if params.Episode > 0 {
		queryParams.Set("ep", strconv.Itoa(params.Episode))
	}

	// Make the request
	resp, err := c.Client.Get(apiURL + "?" + queryParams.Encode())
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	return c.parseResponse(resp.Body)
}

// SearchMovie performs a movie search on the Newznab indexer
func (c *NewznabClient) SearchMovie(params SearchParams) ([]Release, error) {
	if params.Limit == 0 {
		params.Limit = 100
	}

	apiURL := fmt.Sprintf("%s/api", c.BaseURL)

	// Build query parameters
	queryParams := url.Values{}
	queryParams.Set("t", "movie")
	queryParams.Set("apikey", c.APIKey)
	queryParams.Set("limit", strconv.Itoa(params.Limit))
	queryParams.Set("offset", strconv.Itoa(params.Offset))

	// When using imdbid, do NOT send the query parameter as it may cause incorrect results
	// The imdbid is the authoritative identifier
	if params.IMDBID != "" {
		queryParams.Set("imdbid", params.IMDBID)
	} else if params.Query != "" {
		// Only use query if we don't have an imdbid
		queryParams.Set("q", params.Query)
	}

	if len(params.Categories) > 0 {
		queryParams.Set("cat", strings.Join(params.Categories, ","))
	}

	// Make the request
	resp, err := c.Client.Get(apiURL + "?" + queryParams.Encode())
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	return c.parseResponse(resp.Body)
}

// GetRSSFeed gets the latest releases from RSS feed
func (c *NewznabClient) GetRSSFeed(categories []string, limit int) ([]Release, error) {
	if limit == 0 {
		limit = 100
	}

	apiURL := fmt.Sprintf("%s/api", c.BaseURL)

	// Build query parameters
	queryParams := url.Values{}
	queryParams.Set("t", "search")
	queryParams.Set("apikey", c.APIKey)
	queryParams.Set("limit", strconv.Itoa(limit))

	if len(categories) > 0 {
		queryParams.Set("cat", strings.Join(categories, ","))
	}

	// Make the request
	resp, err := c.Client.Get(apiURL + "?" + queryParams.Encode())
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	return c.parseResponse(resp.Body)
}

// TestConnection tests the connection to the Newznab indexer
func (c *NewznabClient) TestConnection() error {
	apiURL := fmt.Sprintf("%s/api", c.BaseURL)

	queryParams := url.Values{}
	queryParams.Set("t", "caps")
	queryParams.Set("apikey", c.APIKey)

	resp, err := c.Client.Get(apiURL + "?" + queryParams.Encode())
	if err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	return nil
}

// parseResponse parses the Newznab XML response
func (c *NewznabClient) parseResponse(reader io.Reader) ([]Release, error) {
	var response NewznabResponse

	decoder := xml.NewDecoder(reader)
	if err := decoder.Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode XML: %w", err)
	}

	releases := make([]Release, 0, len(response.Channel.Items))

	for _, item := range response.Channel.Items {
		release := Release{
			ID:          item.GUID,
			Title:       item.Title,
			GUID:        item.GUID,
			Link:        item.Link,
			Comments:    item.Comments,
			Category:    item.Category,
			Size:        item.Enclosure.Length,
			Description: item.Description,
			DownloadURL: item.Enclosure.URL,
			Attributes:  make(map[string]string),
		}

		// Parse publish date
		if item.PubDate != "" {
			pubDate, err := time.Parse(time.RFC1123Z, item.PubDate)
			if err != nil {
				// Try alternative format
				pubDate, err = time.Parse(time.RFC1123, item.PubDate)
			}
			if err == nil {
				release.PublishDate = pubDate
			}
		}

		// Parse custom attributes
		for _, attr := range item.Attributes {
			release.Attributes[attr.Name] = attr.Value
		}

		releases = append(releases, release)
	}

	return releases, nil
}
