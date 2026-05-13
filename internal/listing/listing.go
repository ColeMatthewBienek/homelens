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

// Fetch pulls full listing detail from Redfin given a /<state>/<city>/<addr>/home/<id> URL.
func Fetch(redfinURL string) (*Detail, error) {
	// Extract property ID from URL
	re := regexp.MustCompile(`/home/(\d+)`)
	m := re.FindStringSubmatch(redfinURL)
	if len(m) < 2 {
		return nil, fmt.Errorf("could not extract property ID from %s", redfinURL)
	}
	propID := m[1]

	u := fmt.Sprintf("https://www.redfin.com/stingray/api/home/details/initialInfo?path=%s&listingVersion=1&accessLevel=1", redfinURL)
	if !strings.Contains(u, "redfin.com") || strings.HasPrefix(redfinURL, "/") {
		// Already a path
		u = fmt.Sprintf("https://www.redfin.com/stingray/api/home/details/initialInfo?path=%s&listingVersion=1&accessLevel=1&propertyId=%s", redfinURL, propID)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	req.Header.Set("Referer", "https://www.redfin.com/")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	body = []byte(strings.TrimPrefix(string(body), "{}&&"))

	var raw map[string]any
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("listing detail parse: %w", err)
	}

	d := &Detail{URL: redfinURL, Raw: raw}
	// Best-effort extraction — Redfin's payload shape is nested and noisy
	if payload, ok := raw["payload"].(map[string]any); ok {
		if pdb, ok := payload["publicRecordsInfo"].(map[string]any); ok {
			if ai, ok := pdb["allInfo"].(map[string]any); ok {
				d.YearBuilt = pickInt(ai, "yearBuilt")
				d.LotSize = pickInt(ai, "lotSize")
			}
		}
		if li, ok := payload["listingInfo"].(map[string]any); ok {
			d.Description = pickString(li, "publicRemarks")
		}
	}
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
