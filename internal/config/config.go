// Package config handles HomeLens configuration: TOML file at
// ~/.config/homelens/config.toml, environment variables (HOMELENS_*),
// profiles, and built-in sane defaults.
//
// Resolution order: CLI flag > env var > active profile > user config > default.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

type Defaults struct {
	MinSqFt    int      `toml:"min_sqft"`
	MaxPrice   int      `toml:"max_price"`
	MinBeds    int      `toml:"min_beds"`
	MinBaths   int      `toml:"min_baths"`
	Types      []string `toml:"types"`
	Status     string   `toml:"status"`
	Theme      string   `toml:"theme"`
	Chunk      int      `toml:"chunk"`
	OutputDir  string   `toml:"output_dir"`
}

type Census struct {
	APIKey string `toml:"api_key"`
}

type Config struct {
	Defaults       Defaults          `toml:"defaults"`
	Census         Census            `toml:"census"`
	ActiveProfile  string            `toml:"active_profile"`
	Profiles       map[string]Defaults `toml:"profiles"`
}

// BuiltInDefaults is what a brand-new user gets before `homelens init`.
func BuiltInDefaults() Defaults {
	return Defaults{
		MinSqFt:   1500,
		MaxPrice:  800000,
		MinBeds:   2,
		MinBaths:  2,
		Types:     []string{"house", "condo", "townhouse"},
		Status:    "for-sale",
		Theme:     "maia",
		Chunk:     25,
		OutputDir: ".",
	}
}

// BuiltInProfiles ships with HomeLens. Users can override or extend.
func BuiltInProfiles() map[string]Defaults {
	return map[string]Defaults{
		"first-home": {
			MaxPrice: 450000, MinBeds: 2, MinBaths: 1, MinSqFt: 1000,
			Types: []string{"house", "condo", "townhouse"}, Status: "for-sale",
		},
		"investment": {
			MinSqFt: 0, MaxPrice: 0, MinBeds: 0, MinBaths: 0,
			Types: []string{"condo", "multi"}, Status: "for-sale",
		},
		"downsize": {
			MinSqFt: 0, MaxPrice: 0, MinBeds: 2, MinBaths: 1,
			Types: []string{"house", "condo", "townhouse"}, Status: "for-sale",
		},
		"luxury": {
			MinSqFt: 3000, MaxPrice: 0, MinBeds: 3, MinBaths: 2,
			Types: []string{"house"}, Status: "for-sale",
		},
	}
}

func configDir() string {
	if d := os.Getenv("HOMELENS_CONFIG_DIR"); d != "" {
		return d
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "homelens")
}

func configPath() string {
	return filepath.Join(configDir(), "config.toml")
}

// Load reads config from disk; returns built-in defaults if file missing.
func Load() (*Config, error) {
	c := &Config{
		Defaults: BuiltInDefaults(),
		Profiles: BuiltInProfiles(),
	}
	path := configPath()
	if _, err := os.Stat(path); err == nil {
		if _, err := toml.DecodeFile(path, c); err != nil {
			return nil, fmt.Errorf("decode %s: %w", path, err)
		}
	}
	// Merge built-in profiles if user didn't define them
	if c.Profiles == nil {
		c.Profiles = BuiltInProfiles()
	} else {
		for k, v := range BuiltInProfiles() {
			if _, ok := c.Profiles[k]; !ok {
				c.Profiles[k] = v
			}
		}
	}
	// Apply env vars (overrides anything from disk)
	applyEnv(&c.Defaults)
	// Try to read Census key from legacy census-pp-cli location if not in our config
	if c.Census.APIKey == "" {
		c.Census.APIKey = readLegacyCensusKey()
	}
	return c, nil
}

// Save writes config back to disk.
func Save(c *Config) error {
	if err := os.MkdirAll(configDir(), 0700); err != nil {
		return err
	}
	f, err := os.OpenFile(configPath(), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := toml.NewEncoder(f)
	return enc.Encode(c)
}

// ResolveProfile applies a named profile on top of base defaults, then env.
// Empty fields in the profile fall through to the base.
func ResolveProfile(c *Config, profileName string) Defaults {
	base := c.Defaults
	if profileName == "" {
		profileName = c.ActiveProfile
	}
	if profileName == "" {
		return base
	}
	p, ok := c.Profiles[profileName]
	if !ok {
		return base
	}
	if p.MinSqFt > 0 {
		base.MinSqFt = p.MinSqFt
	}
	if p.MaxPrice > 0 {
		base.MaxPrice = p.MaxPrice
	}
	if p.MinBeds > 0 {
		base.MinBeds = p.MinBeds
	}
	if p.MinBaths > 0 {
		base.MinBaths = p.MinBaths
	}
	if len(p.Types) > 0 {
		base.Types = p.Types
	}
	if p.Status != "" {
		base.Status = p.Status
	}
	if p.Theme != "" {
		base.Theme = p.Theme
	}
	return base
}

func applyEnv(d *Defaults) {
	if v := os.Getenv("HOMELENS_MIN_SQFT"); v != "" {
		fmt.Sscanf(v, "%d", &d.MinSqFt)
	}
	if v := os.Getenv("HOMELENS_MAX_PRICE"); v != "" {
		fmt.Sscanf(v, "%d", &d.MaxPrice)
	}
	if v := os.Getenv("HOMELENS_MIN_BEDS"); v != "" {
		fmt.Sscanf(v, "%d", &d.MinBeds)
	}
	if v := os.Getenv("HOMELENS_MIN_BATHS"); v != "" {
		fmt.Sscanf(v, "%d", &d.MinBaths)
	}
	if v := os.Getenv("HOMELENS_TYPES"); v != "" {
		d.Types = strings.Split(v, ",")
	}
	if v := os.Getenv("HOMELENS_THEME"); v != "" {
		d.Theme = v
	}
	if v := os.Getenv("HOMELENS_OUTPUT_DIR"); v != "" {
		d.OutputDir = v
	}
}

func readLegacyCensusKey() string {
	home, _ := os.UserHomeDir()
	path := filepath.Join(home, ".config", "census-pp-cli", "config.toml")
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "api_key") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				return strings.Trim(strings.TrimSpace(parts[1]), `"'`)
			}
		}
	}
	return ""
}

// TypesToUIPT converts user-friendly type names to Redfin uiPropertyType ints.
func TypesToUIPT(types []string) []int {
	m := map[string]int{
		"house": 1, "condo": 2, "townhouse": 3,
		"multi": 4, "multi-family": 4, "land": 5,
	}
	var out []int
	for _, t := range types {
		t = strings.ToLower(strings.TrimSpace(t))
		if v, ok := m[t]; ok {
			out = append(out, v)
		}
	}
	if len(out) == 0 {
		out = []int{1, 2, 3}
	}
	return out
}

// ConfigPath returns the absolute path to the user config file.
func ConfigPath() string { return configPath() }
