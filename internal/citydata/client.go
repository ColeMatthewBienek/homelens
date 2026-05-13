// Package citydata scrapes zip-code-level demographics from city-data.com.
// No API key — the data is rendered into the HTML; we extract with regex.
//
// If city-data.com changes its HTML structure (it does, every ~year), the
// regexes here need updating. v0 uses the structure as of May 2026.
package citydata

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type ZipProfile struct {
	Zip                    string  `json:"zip"`
	TotalPopulation        int     `json:"total_population"`
	MedianAge              float64 `json:"median_age"`
	MedianHouseholdIncome  int     `json:"median_household_income"`
	MedianHouseValue       int     `json:"median_house_value"`
	PctBachelorsOrHigher   float64 `json:"pct_bachelors_or_higher"`
	PovertyPct             float64 `json:"poverty_pct"`
	WhitePct               float64 `json:"white_pct"`
	BlackPct               float64 `json:"black_pct"`
	HispanicPct            float64 `json:"hispanic_pct"`
	AsianPct               float64 `json:"asian_pct"`
	OtherPct               float64 `json:"other_pct"`
}

// FetchZip pulls the city-data.com ZIP page and parses the demographic stats.
// On parse failure for individual fields, the field stays zero — callers
// should treat zero as "unknown" rather than "actually zero".
//
// If city-data-pp-cli is on PATH, HomeLens delegates to it.
func FetchZip(zip string) (*ZipProfile, error) {
	if path, err := exec.LookPath("city-data-pp-cli"); err == nil {
		if out, err := exec.Command(path, "zip", zip).Output(); err == nil {
			var p ZipProfile
			if err := json.Unmarshal(out, &p); err == nil {
				return &p, nil
			}
		}
	}
	u := "https://www.city-data.com/zips/" + zip + ".html"
	body, err := fetch(u)
	if err != nil {
		return nil, err
	}
	p := &ZipProfile{Zip: zip}
	p.TotalPopulation = parseIntMatch(body, `Total population.*?(\d[\d,]+)`)
	p.MedianAge = parseFloatMatch(body, `Median resident age.*?(\d+\.\d+)`)
	p.MedianHouseholdIncome = parseIntMatch(body, `Estimated median household income[^$]*\$(\d[\d,]+)`)
	p.MedianHouseValue = parseIntMatch(body, `Estimated median house.*?value[^$]*\$(\d[\d,]+)`)
	p.PctBachelorsOrHigher = parseFloatMatch(body, `[Bb]achelor.*?(\d+\.\d+)%`)
	p.PovertyPct = parseFloatMatch(body, `[Bb]elow poverty.*?(\d+\.\d+)%`)
	p.WhitePct = parseFloatMatch(body, `White alone.*?(\d+\.\d+)%`)
	p.BlackPct = parseFloatMatch(body, `Black.*?alone.*?(\d+\.\d+)%`)
	p.HispanicPct = parseFloatMatch(body, `Hispanic.*?(\d+\.\d+)%`)
	p.AsianPct = parseFloatMatch(body, `Asian alone.*?(\d+\.\d+)%`)
	p.OtherPct = 100 - p.WhitePct - p.BlackPct - p.HispanicPct - p.AsianPct
	if p.OtherPct < 0 {
		p.OtherPct = 0
	}
	return p, nil
}

type CityCrime struct {
	City     string  `json:"city"`
	State    string  `json:"state"`
	Index    float64 `json:"index"`
	USAvg    float64 `json:"us_avg"`
	Year     int     `json:"year"`
}

// FetchCityCrime takes a city-data slug like "Vancouver-Washington" and
// returns the city-wide crime index plus US average baseline.
func FetchCityCrime(slug string) (*CityCrime, error) {
	u := "https://www.city-data.com/city/" + slug + ".html"
	body, err := fetch(u)
	if err != nil {
		return nil, err
	}
	c := &CityCrime{}
	// Parse "City-data.com crime index" line
	c.Index = parseFloatMatch(body, `[Cc]rime index in [^:<]*?:\s*(\d+\.\d+)`)
	c.USAvg = parseFloatMatch(body, `[Uu]\.?S\.? average.*?(\d+\.\d+)`)
	if reYear := regexp.MustCompile(`(?:in|for)\s+(20\d{2})`); reYear != nil {
		if m := reYear.FindStringSubmatch(body); len(m) > 1 {
			c.Year, _ = strconv.Atoi(m[1])
		}
	}
	parts := strings.SplitN(slug, "-", 2)
	if len(parts) == 2 {
		c.City = parts[0]
		c.State = parts[1]
	}
	return c, nil
}

func fetch(u string) (string, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (homelens/0.1)")
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("city-data: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("city-data HTTP %d for %s", resp.StatusCode, u)
	}
	b, err := io.ReadAll(resp.Body)
	return string(b), err
}

func parseIntMatch(body, pat string) int {
	re := regexp.MustCompile(pat)
	m := re.FindStringSubmatch(body)
	if len(m) < 2 {
		return 0
	}
	clean := strings.ReplaceAll(m[1], ",", "")
	n, _ := strconv.Atoi(clean)
	return n
}

func parseFloatMatch(body, pat string) float64 {
	re := regexp.MustCompile(pat)
	m := re.FindStringSubmatch(body)
	if len(m) < 2 {
		return 0
	}
	f, _ := strconv.ParseFloat(m[1], 64)
	return f
}
