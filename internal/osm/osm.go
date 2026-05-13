// Package osm queries the OpenStreetMap Overpass API to count nearby
// amenities and derive a walkability-style composite score (0-100).
//
// Keyless, generous rate limits. Used by the listing deep-dive.
package osm

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Amenities struct {
	Restaurants int `json:"restaurants"`
	Cafes       int `json:"cafes"`
	Grocery     int `json:"grocery"`
	Schools     int `json:"schools"`
	Parks       int `json:"parks"`
	Transit     int `json:"transit"`
	Pharmacies  int `json:"pharmacies"`
	Hospitals   int `json:"hospitals"`
	Gyms        int `json:"gyms"`
	Score       int `json:"score"` // 0-100 composite (Walk Score-style)
}

// Fetch counts amenities within radiusMeters of (lat, lng).
func Fetch(lat, lng float64, radiusMeters int) (*Amenities, error) {
	if radiusMeters == 0 {
		radiusMeters = 1609 // ~1 mile
	}
	q := fmt.Sprintf(`[out:json][timeout:25];
(
  node["amenity"="restaurant"](around:%d,%f,%f);
  node["amenity"="cafe"](around:%d,%f,%f);
  node["shop"~"supermarket|convenience"](around:%d,%f,%f);
  node["amenity"~"school|kindergarten|college|university"](around:%d,%f,%f);
  way["leisure"="park"](around:%d,%f,%f);
  node["public_transport"="stop_position"](around:%d,%f,%f);
  node["amenity"="pharmacy"](around:%d,%f,%f);
  node["amenity"~"hospital|clinic"](around:%d,%f,%f);
  node["leisure"~"fitness_centre|sports_centre"](around:%d,%f,%f);
);
out tags;`,
		radiusMeters, lat, lng,
		radiusMeters, lat, lng,
		radiusMeters, lat, lng,
		radiusMeters, lat, lng,
		radiusMeters, lat, lng,
		radiusMeters, lat, lng,
		radiusMeters, lat, lng,
		radiusMeters, lat, lng,
		radiusMeters, lat, lng,
	)

	form := url.Values{}
	form.Set("data", q)
	req, _ := http.NewRequest("POST", "https://overpass-api.de/api/interpreter", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "homelens/0.3")

	client := &http.Client{Timeout: 45 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("overpass: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var r struct {
		Elements []struct {
			Tags map[string]string `json:"tags"`
		} `json:"elements"`
	}
	if err := json.Unmarshal(body, &r); err != nil {
		return nil, err
	}

	a := &Amenities{}
	for _, el := range r.Elements {
		switch {
		case el.Tags["amenity"] == "restaurant":
			a.Restaurants++
		case el.Tags["amenity"] == "cafe":
			a.Cafes++
		case el.Tags["shop"] == "supermarket" || el.Tags["shop"] == "convenience":
			a.Grocery++
		case el.Tags["amenity"] == "school" || el.Tags["amenity"] == "kindergarten" ||
			el.Tags["amenity"] == "college" || el.Tags["amenity"] == "university":
			a.Schools++
		case el.Tags["leisure"] == "park":
			a.Parks++
		case el.Tags["public_transport"] == "stop_position":
			a.Transit++
		case el.Tags["amenity"] == "pharmacy":
			a.Pharmacies++
		case el.Tags["amenity"] == "hospital" || el.Tags["amenity"] == "clinic":
			a.Hospitals++
		case el.Tags["leisure"] == "fitness_centre" || el.Tags["leisure"] == "sports_centre":
			a.Gyms++
		}
	}

	// Composite score: weighted log-ish bins approximating Walk Score
	a.Score = composite(a)
	return a, nil
}

func composite(a *Amenities) int {
	// Each category capped at ~25-30 weight, log-decay past saturation
	wt := func(n, sat, weight int) int {
		if n <= 0 {
			return 0
		}
		if n >= sat {
			return weight
		}
		return n * weight / sat
	}
	s := 0
	s += wt(a.Restaurants+a.Cafes, 20, 25) // dining
	s += wt(a.Grocery, 3, 20)
	s += wt(a.Transit, 8, 15)
	s += wt(a.Schools, 4, 10)
	s += wt(a.Parks, 4, 10)
	s += wt(a.Pharmacies, 2, 8)
	s += wt(a.Gyms, 2, 7)
	s += wt(a.Hospitals, 1, 5)
	if s > 100 {
		s = 100
	}
	return s
}
