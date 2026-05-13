package redfin

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// ResolveCity takes "Vancouver, WA" or "Austin, TX" and returns a Redfin
// region slug like "city/18823/WA/Vancouver". Uses DuckDuckGo HTML results
// because Redfin's autocomplete is bot-blocked.
//
// Returns the numeric region ID; caller can pass it directly to Search.
func ResolveCity(query string) (string, error) {
	q := url.QueryEscape(strings.TrimSpace(query) + " redfin homes for sale")
	u := "https://html.duckduckgo.com/html/?q=" + q

	client := &http.Client{Timeout: 20 * time.Second}
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36")
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("city resolve: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Look for redfin.com/city/<id>/<ST>/<Name>
	re := regexp.MustCompile(`redfin\.com/city/(\d+)/([A-Z]{2})/([^"'/&?]+)`)
	m := re.FindStringSubmatch(string(body))
	if len(m) < 4 {
		return "", fmt.Errorf("could not find Redfin city slug for %q", query)
	}
	return "city/" + m[1] + "/" + m[2] + "/" + m[3], nil
}
