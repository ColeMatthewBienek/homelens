// Package store persists saved searches and watch history under
// ~/.config/homelens/{searches,history}/. Searches are TOML, history is JSON.
package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/ColeMatthewBienek/homelens/internal/redfin"
)

type SavedSearch struct {
	Name      string   `toml:"name"`
	Location  string   `toml:"location"`
	Slug      string   `toml:"slug,omitempty"`
	MaxPrice  int      `toml:"max_price"`
	MinBeds   int      `toml:"min_beds"`
	MinBaths  int      `toml:"min_baths"`
	MinSqFt   int      `toml:"min_sqft"`
	Types     []string `toml:"types"`
	Status    string   `toml:"status"`
	Theme     string   `toml:"theme"`
	CreatedAt string   `toml:"created_at"`
}

func searchDir() string {
	home, _ := os.UserHomeDir()
	if d := os.Getenv("HOMELENS_CONFIG_DIR"); d != "" {
		return filepath.Join(d, "searches")
	}
	return filepath.Join(home, ".config", "homelens", "searches")
}

func historyDir(name string) string {
	home, _ := os.UserHomeDir()
	base := filepath.Join(home, ".config", "homelens", "history")
	if d := os.Getenv("HOMELENS_CONFIG_DIR"); d != "" {
		base = filepath.Join(d, "history")
	}
	return filepath.Join(base, name)
}

func SaveSearch(s SavedSearch) error {
	if err := os.MkdirAll(searchDir(), 0700); err != nil {
		return err
	}
	if s.CreatedAt == "" {
		s.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	f, err := os.Create(filepath.Join(searchDir(), s.Name+".toml"))
	if err != nil {
		return err
	}
	defer f.Close()
	return toml.NewEncoder(f).Encode(s)
}

func LoadSearch(name string) (*SavedSearch, error) {
	var s SavedSearch
	path := filepath.Join(searchDir(), name+".toml")
	if _, err := toml.DecodeFile(path, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func ListSearches() ([]string, error) {
	entries, err := os.ReadDir(searchDir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".toml" {
			out = append(out, e.Name()[:len(e.Name())-5])
		}
	}
	return out, nil
}

// SaveSnapshot writes the current results of a watched search to history
// for later diffing. Filename is the timestamp.
func SaveSnapshot(searchName string, homes []redfin.Home) (string, error) {
	dir := historyDir(searchName)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	stamp := time.Now().UTC().Format("20060102T150405")
	path := filepath.Join(dir, stamp+".json")
	data, err := json.MarshalIndent(homes, "", "  ")
	if err != nil {
		return "", err
	}
	return path, os.WriteFile(path, data, 0600)
}

// LatestSnapshot returns the most recent prior snapshot, or nil if none.
// TODO(watch): wire diff logic against prior snapshot.
func LatestSnapshot(searchName string) ([]redfin.Home, string, error) {
	dir := historyDir(searchName)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, "", err
	}
	if len(entries) == 0 {
		return nil, "", nil
	}
	// Sort lexicographically — timestamp filenames sort chronologically
	var latest string
	for _, e := range entries {
		if e.Name() > latest {
			latest = e.Name()
		}
	}
	data, err := os.ReadFile(filepath.Join(dir, latest))
	if err != nil {
		return nil, "", err
	}
	var homes []redfin.Home
	if err := json.Unmarshal(data, &homes); err != nil {
		return nil, "", err
	}
	return homes, latest, nil
}
