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
	"os/exec"
	"strconv"
	"time"
)

type GeocodeResult struct {
	State   string `json:"state"`    // 2-digit FIPS state
	County  string `json:"county"`   // 3-digit FIPS county
	Tract   string `json:"tract"`    // 6-digit tract
	TractID string `json:"tract_id"` // human-readable, e.g. "Census Tract 407.09"
}

// Geocode resolves lat/lng to a census tract via the Census Geocoder.
//
// If census-pp-cli is on PATH (a printing-press dependency CLI), HomeLens
// delegates to it. Otherwise it calls the Geocoder API directly. Either way
// the result shape is identical.
func Geocode(lat, lng float64) (*GeocodeResult, error) {
	if path, err := exec.LookPath("census-pp-cli"); err == nil {
		if r, err := geocodeViaCLI(path, lat, lng); err == nil {
			return r, nil
		}
		// fall through to inline on CLI error
	}
	return geocodeInline(lat, lng)
}

func geocodeViaCLI(binPath string, lat, lng float64) (*GeocodeResult, error) {
	out, err := exec.Command(binPath, "geocode", strconv.FormatFloat(lat, 'f', -1, 64), strconv.FormatFloat(lng, 'f', -1, 64)).Output()
	if err != nil {
		return nil, err
	}
	var r GeocodeResult
	if err := json.Unmarshal(out, &r); err != nil {
		return nil, err
	}
	return &r, nil
}

func geocodeInline(lat, lng float64) (*GeocodeResult, error) {
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
