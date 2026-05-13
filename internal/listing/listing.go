// Package listing handles the single-listing deep-dive view.
package listing

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// Detail wraps Redfin's /stingray/api/home/details/ payload (a subset).
type Detail struct {
	Address        string
	City, State    string
	Zip            string
	Price          int
	Beds           float64
	Baths          float64
	SqFt           int
	YearBuilt      int
	LotSize        int
	HOA            int
	DOM            int
	PricePerSqFt   int
	URL            string
	Description    string
	Schools        []School
	Latitude       float64
	Longitude      float64
	Raw            map[string]any `json:"-"`
}

type School struct {
	Name   string  `json:"name"`
	Grade  string  `json:"grade"`
	Rating float64 `json:"rating"`
}

// Fetch pulls full listing detail from Redfin given a /<state>/<city>/<addr>/home/<id>
// URL or full https URL. Scrapes the public HTML page for lat/lng/description because
// the Stingray detail API requires session cookies; the HTML page is reliably anonymous.
func Fetch(redfinURL string) (*Detail, error) {
	// Normalize to a full URL
	full := redfinURL
	if strings.HasPrefix(full, "/") {
		full = "https://www.redfin.com" + full
	}

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", full, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	html := string(body)

	d := &Detail{URL: redfinURL}

	// lat/lng appear in inlined JSON like "latitude":45.6543,"longitude":-122.6543
	reLat := regexp.MustCompile(`"latitude"\s*:\s*([\-\d.]+)`)
	reLng := regexp.MustCompile(`"longitude"\s*:\s*([\-\d.]+)`)
	if m := reLat.FindStringSubmatch(html); len(m) > 1 {
		fmt.Sscanf(m[1], "%f", &d.Latitude)
	}
	if m := reLng.FindStringSubmatch(html); len(m) > 1 {
		fmt.Sscanf(m[1], "%f", &d.Longitude)
	}
	reYear := regexp.MustCompile(`"yearBuilt"[^}]*?"value"\s*:\s*(\d{4})`)
	if m := reYear.FindStringSubmatch(html); len(m) > 1 {
		fmt.Sscanf(m[1], "%d", &d.YearBuilt)
	}
	reLot := regexp.MustCompile(`"lotSize"[^}]*?"value"\s*:\s*(\d+)`)
	if m := reLot.FindStringSubmatch(html); len(m) > 1 {
		fmt.Sscanf(m[1], "%d", &d.LotSize)
	}
	// Public remarks / description (og:description is a clean source)
	reDesc := regexp.MustCompile(`<meta\s+property="og:description"\s+content="([^"]+)"`)
	if m := reDesc.FindStringSubmatch(html); len(m) > 1 {
		// HTML-decode common entities
		d.Description = strings.NewReplacer("&quot;", `"`, "&amp;", "&", "&lt;", "<", "&gt;", ">", "&#39;", "'").Replace(m[1])
	}

	// Try the JSON detail API as a best-effort enrichment (often blocked, that's OK)
	rePid := regexp.MustCompile(`/home/(\d+)`)
	if m := rePid.FindStringSubmatch(redfinURL); len(m) > 1 {
		_ = m // could call stingray/api/home/details if logged in; skip for now
	}
	_ = json.Unmarshal
	return d, nil
}

func pickInt(m map[string]any, key string) int {
	if v, ok := m[key]; ok {
		switch t := v.(type) {
		case float64:
			return int(t)
		case int:
			return t
		}
	}
	return 0
}

func pickString(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
