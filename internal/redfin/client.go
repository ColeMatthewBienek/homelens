// Package redfin queries the Redfin Stingray search API.
//
// This client uses the direct-curl workaround documented in the user's memory
// (feedback_redfin_query_workflow.md) because the upstream printed redfin-pp-cli
// drops the required al=1 arg. When upstream lands the fix, only this file
// needs updating.
package redfin

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Filters struct {
	MaxPrice int
	MinBeds  int
	MinBaths int
	MinSqFt  int
	Types    []int // uiPropertyType: 1=house, 2=condo, 3=townhouse, 4=multi, 5=land
	Status   int   // 1=for-sale, 9=sold, etc.
}

type ValueWrap struct {
	Value json.RawMessage `json:"value"`
	Level int             `json:"level,omitempty"`
}

type Home struct {
	URL            string    `json:"url"`
	Price          ValueWrap `json:"price"`
	Beds           float64   `json:"beds"`
	Baths          float64   `json:"baths"`
	SqFt           ValueWrap `json:"sqFt"`
	PricePerSqFt   ValueWrap `json:"pricePerSqFt"`
	UIPropertyType int       `json:"uiPropertyType"`
	Zip            string    `json:"zip"`
	City           string    `json:"city"`
	State          string    `json:"state"`
	StreetLine     ValueWrap `json:"streetLine"`
	UnitNumber     ValueWrap `json:"unitNumber"`
	YearBuilt      ValueWrap `json:"yearBuilt"`
	DOM            ValueWrap `json:"dom"`
	LotSize        ValueWrap `json:"lotSize"`
	HOA            ValueWrap `json:"hoa"`
	MLSID          ValueWrap `json:"mlsId"`
	Latitude       ValueWrap `json:"latitude"`
	Longitude      ValueWrap `json:"longitude"`
	ListingBroker  struct {
		Name string `json:"name"`
	} `json:"listingBroker"`
	ListingAgent struct {
		Name string `json:"name"`
	} `json:"listingAgent"`
	Sashes []struct {
		SashTypeName string `json:"sashTypeName"`
	} `json:"sashes"`
	ListingTags []string `json:"listingTags"`
	KeyFacts    []struct {
		Description string `json:"description"`
	} `json:"keyFacts"`
	PhotoURL string `json:"-"` // filled by listing photo lookup if available
}

type response struct {
	Payload struct {
		Homes []Home `json:"homes"`
	} `json:"payload"`
}

// Helpers to extract typed values from the Redfin envelope.
func (v ValueWrap) Int() int {
	if len(v.Value) == 0 {
		return 0
	}
	var f float64
	if err := json.Unmarshal(v.Value, &f); err == nil {
		return int(f)
	}
	var s string
	if err := json.Unmarshal(v.Value, &s); err == nil {
		n, _ := strconv.Atoi(s)
		return n
	}
	return 0
}

func (v ValueWrap) Float() float64 {
	if len(v.Value) == 0 {
		return 0
	}
	var f float64
	if err := json.Unmarshal(v.Value, &f); err == nil {
		return f
	}
	return 0
}

func (v ValueWrap) String() string {
	if len(v.Value) == 0 {
		return ""
	}
	var s string
	if err := json.Unmarshal(v.Value, &s); err == nil {
		return s
	}
	var f float64
	if err := json.Unmarshal(v.Value, &f); err == nil {
		return strconv.FormatFloat(f, 'f', -1, 64)
	}
	return ""
}

func (v ValueWrap) Present() bool {
	return len(v.Value) > 0 && string(v.Value) != "null"
}

// Search queries Redfin's stingray/api/gis endpoint with the required al=1
// arg and paginates `pages` times, then dedupes and applies client-side
// filter floors (the server doesn't strictly honor min-sqft / min-beds).
func Search(regionSlug string, f Filters, pages int) ([]Home, error) {
	regionID, err := extractRegionID(regionSlug)
	if err != nil {
		return nil, err
	}

	if f.Status == 0 {
		f.Status = 1
	}
	if pages <= 0 {
		pages = 3
	}
	if len(f.Types) == 0 {
		f.Types = []int{1, 2, 3}
	}

	uiptStrs := make([]string, len(f.Types))
	for i, t := range f.Types {
		uiptStrs[i] = strconv.Itoa(t)
	}
	uipt := strings.Join(uiptStrs, ",")

	client := &http.Client{Timeout: 30 * time.Second}
	var all []Home
	for p := 1; p <= pages; p++ {
		q := url.Values{}
		q.Set("al", "1")
		if f.MaxPrice > 0 {
			q.Set("max_price", strconv.Itoa(f.MaxPrice))
		}
		if f.MinBeds > 0 {
			q.Set("min_beds", strconv.Itoa(f.MinBeds))
		}
		if f.MinBaths > 0 {
			q.Set("min_baths", strconv.Itoa(f.MinBaths))
		}
		if f.MinSqFt > 0 {
			q.Set("min_sqft", strconv.Itoa(f.MinSqFt))
		}
		q.Set("num_homes", "50")
		q.Set("page_number", strconv.Itoa(p))
		q.Set("region_id", regionID)
		q.Set("region_type", "6")
		q.Set("status", strconv.Itoa(f.Status))
		q.Set("uipt", uipt)
		q.Set("v", "8")

		u := "https://www.redfin.com/stingray/api/gis?" + q.Encode()
		req, err := http.NewRequest("GET", u, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
		req.Header.Set("Referer", "https://www.redfin.com/")

		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("redfin page %d: %w", p, err)
		}
		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != 200 {
			return nil, fmt.Errorf("redfin page %d: HTTP %d", p, resp.StatusCode)
		}
		// Strip CSRF prefix
		body = []byte(strings.TrimPrefix(string(body), "{}&&"))
		var r response
		if err := json.Unmarshal(body, &r); err != nil {
			return nil, fmt.Errorf("redfin page %d json: %w", p, err)
		}
		all = append(all, r.Payload.Homes...)
		if len(r.Payload.Homes) < 50 {
			break // last page
		}
	}

	// Dedupe by URL
	seen := map[string]bool{}
	var out []Home
	for _, h := range all {
		if seen[h.URL] {
			continue
		}
		seen[h.URL] = true

		// Client-side filter enforcement
		if f.MinBeds > 0 && int(h.Beds) < f.MinBeds {
			continue
		}
		if f.MinBaths > 0 && h.Baths < float64(f.MinBaths) {
			continue
		}
		if f.MinSqFt > 0 && h.SqFt.Int() < f.MinSqFt {
			continue
		}
		if f.MaxPrice > 0 && h.Price.Int() > f.MaxPrice {
			continue
		}
		if len(f.Types) > 0 {
			ok := false
			for _, t := range f.Types {
				if h.UIPropertyType == t {
					ok = true
					break
				}
			}
			if !ok {
				continue
			}
		}
		out = append(out, h)
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].Price.Int() < out[j].Price.Int()
	})

	return out, nil
}

// extractRegionID accepts either "18823" or "city/18823/WA/Vancouver" and
// returns just the numeric ID.
func extractRegionID(slug string) (string, error) {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return "", fmt.Errorf("empty region slug")
	}
	if _, err := strconv.Atoi(slug); err == nil {
		return slug, nil
	}
	parts := strings.Split(slug, "/")
	for _, p := range parts {
		if _, err := strconv.Atoi(p); err == nil {
			return p, nil
		}
	}
	return "", fmt.Errorf("could not extract region ID from slug %q", slug)
}
