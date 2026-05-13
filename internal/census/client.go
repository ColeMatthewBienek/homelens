// Package census wraps the keyless US Census Geocoder API (for lat/lng →
// census tract resolution) and the keyed ACS 5-year API for tract-level
// demographics. The ACS key is read from ~/.config/homelens/config.toml or
// the HOMELENS_CENSUS_KEY env var; if missing, tract enrichment is skipped
// gracefully.
package census

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type GeocodeResult struct {
	State   string // 2-digit FIPS state code
	County  string // 3-digit FIPS county code
	Tract   string // 6-digit census tract code
	TractID string // human-readable tract number (e.g. "407.09")
}

// Geocode resolves lat/lng to a census tract via the Census Geocoder.
// Always passes -L equivalent (Go's http.Client follows redirects by default).
func Geocode(lat, lng float64) (*GeocodeResult, error) {
	q := url.Values{}
	q.Set("x", fmt.Sprintf("%f", lng))
	q.Set("y", fmt.Sprintf("%f", lat))
	q.Set("benchmark", "Public_AR_Current")
	q.Set("vintage", "Current_Current")
	q.Set("format", "json")

	u := "https://geocoding.geo.census.gov/geocoder/geographies/coordinates?" + q.Encode()
	client := &http.Client{Timeout: 20 * time.Second}
	resp, err := client.Get(u)
	if err != nil {
		return nil, fmt.Errorf("census geocode: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var r struct {
		Result struct {
			Geographies struct {
				CensusTracts []struct {
					State   string `json:"STATE"`
					County  string `json:"COUNTY"`
					Tract   string `json:"TRACT"`
					Name    string `json:"NAME"`
				} `json:"Census Tracts"`
			} `json:"geographies"`
		} `json:"result"`
	}
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, fmt.Errorf("census geocode json: %w", err)
	}
	if len(r.Result.Geographies.CensusTracts) == 0 {
		return nil, fmt.Errorf("no census tract at %f,%f", lat, lng)
	}
	t := r.Result.Geographies.CensusTracts[0]
	return &GeocodeResult{
		State:   t.State,
		County:  t.County,
		Tract:   t.Tract,
		TractID: t.Name,
	}, nil
}
