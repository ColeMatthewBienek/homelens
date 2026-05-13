package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ColeMatthewBienek/homelens/internal/census"
	"github.com/ColeMatthewBienek/homelens/internal/compare"
	"github.com/ColeMatthewBienek/homelens/internal/config"
	"github.com/ColeMatthewBienek/homelens/internal/diff"
	"github.com/ColeMatthewBienek/homelens/internal/listing"
	"github.com/ColeMatthewBienek/homelens/internal/osm"
	"github.com/ColeMatthewBienek/homelens/internal/share"
	"github.com/ColeMatthewBienek/homelens/internal/store"
)

func initCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Interactive bootstrap — writes ~/.config/homelens/config.toml",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			rd := bufio.NewReader(os.Stdin)
			fmt.Println("HomeLens init — press Enter to keep the shown default.")
			fmt.Println()

			ask := func(prompt, def string) string {
				fmt.Printf("%s [%s]: ", prompt, def)
				line, _ := rd.ReadString('\n')
				line = strings.TrimSpace(line)
				if line == "" {
					return def
				}
				return line
			}
			askInt := func(prompt string, def int) int {
				s := ask(prompt, fmt.Sprintf("%d", def))
				var n int
				if _, err := fmt.Sscanf(s, "%d", &n); err != nil {
					return def
				}
				return n
			}

			cfg.Defaults.MinSqFt = askInt("Minimum square feet", cfg.Defaults.MinSqFt)
			cfg.Defaults.MaxPrice = askInt("Maximum price", cfg.Defaults.MaxPrice)
			cfg.Defaults.MinBeds = askInt("Minimum bedrooms", cfg.Defaults.MinBeds)
			cfg.Defaults.MinBaths = askInt("Minimum bathrooms", cfg.Defaults.MinBaths)
			types := ask("Property types (house,condo,townhouse,multi,land)", strings.Join(cfg.Defaults.Types, ","))
			cfg.Defaults.Types = strings.Split(types, ",")
			cfg.Defaults.Theme = ask("Default theme (bloom, modern, classic, minimal, dark)", cfg.Defaults.Theme)
			cfg.Defaults.OutputDir = ask("Default output directory", cfg.Defaults.OutputDir)

			if cfg.Census.APIKey == "" {
				key := ask("Census API key (free at https://api.census.gov/data/key_signup.html; Enter to skip)", "")
				if key != "" {
					cfg.Census.APIKey = key
				}
			}

			if err := config.Save(cfg); err != nil {
				return err
			}
			fmt.Println()
			fmt.Println("✓ Saved to", config.ConfigPath())
			return nil
		},
	}
}

func saveCmd() *cobra.Command {
	var slug, types, theme string
	var maxPrice, minBeds, minBaths, minSqft int
	cmd := &cobra.Command{
		Use:   "save <name> <location>",
		Short: "Save a named search",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			loc := strings.Join(args[1:], " ")
			s := store.SavedSearch{
				Name: name, Location: loc, Slug: slug,
				MaxPrice: maxPrice, MinBeds: minBeds, MinBaths: minBaths, MinSqFt: minSqft,
				Status: "for-sale", Theme: theme,
			}
			if types != "" {
				s.Types = strings.Split(types, ",")
			}
			if err := store.SaveSearch(s); err != nil {
				return err
			}
			fmt.Printf("✓ Saved search %q (location: %s)\n", name, loc)
			fmt.Printf("Run with:  homelens-pp-cli search %s\n", name)
			return nil
		},
	}
	cmd.Flags().StringVar(&slug, "slug", "", "Redfin slug")
	cmd.Flags().StringVar(&types, "types", "", "house,condo,townhouse")
	cmd.Flags().StringVar(&theme, "theme", "", "report theme")
	cmd.Flags().IntVar(&maxPrice, "max-price", 0, "max price")
	cmd.Flags().IntVar(&minBeds, "min-beds", 0, "min beds")
	cmd.Flags().IntVar(&minBaths, "min-baths", 0, "min baths")
	cmd.Flags().IntVar(&minSqft, "min-sqft", 0, "min sqft")
	return cmd
}

func listSearchesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list-searches",
		Short: "List all saved searches",
		RunE: func(cmd *cobra.Command, args []string) error {
			names, err := store.ListSearches()
			if err != nil {
				return err
			}
			if len(names) == 0 {
				fmt.Println("No saved searches yet. Use `homelens-pp-cli save <name> <location>`.")
				return nil
			}
			for _, n := range names {
				s, err := store.LoadSearch(n)
				if err != nil {
					continue
				}
				fmt.Printf("• %s — %s\n", n, s.Location)
			}
			return nil
		},
	}
}

func watchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "watch <saved-search-name>",
		Short: "Re-run a saved search and diff against last run",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			saved, err := store.LoadSearch(name)
			if err != nil {
				return fmt.Errorf("no saved search %q (run `homelens-pp-cli list-searches`)", name)
			}
			cfg, _ := config.Load()
			opts := SearchOpts{
				Location: saved.Location,
				Slug:     saved.Slug,
				MaxPrice: nz(saved.MaxPrice, cfg.Defaults.MaxPrice),
				MinBeds:  nz(saved.MinBeds, cfg.Defaults.MinBeds),
				MinBaths: nz(saved.MinBaths, cfg.Defaults.MinBaths),
				MinSqFt:  nz(saved.MinSqFt, cfg.Defaults.MinSqFt),
				Types:    saved.Types,
				NoEnrich: true,
			}
			if len(opts.Types) == 0 {
				opts.Types = cfg.Defaults.Types
			}
			res, err := runSearch(opts)
			if err != nil {
				return err
			}
			prev, _, _ := store.LatestSnapshot(name)
			d := diff.Compute(prev, res.Homes)
			snap, _ := store.SaveSnapshot(name, res.Homes)
			fmt.Printf("Snapshot: %s\n", snap)
			fmt.Printf("Current: %d listings · prior snapshot: %d listings · unchanged: %d\n", len(res.Homes), len(prev), d.Unchanged)
			fmt.Printf("New: %d · Removed: %d · Price-changed: %d\n", len(d.New), len(d.Removed), len(d.Changed))
			for _, c := range d.New {
				fmt.Printf("  + NEW       $%d · %s\n", c.NewPrice, c.Address)
			}
			for _, c := range d.Removed {
				fmt.Printf("  - REMOVED   $%d · %s\n", c.OldPrice, c.Address)
			}
			for _, c := range d.Changed {
				delta := c.NewPrice - c.OldPrice
				sign := "+"
				if delta < 0 {
					sign = ""
				}
				fmt.Printf("  ~ %s%-9d $%d → $%d · %s\n", sign, delta, c.OldPrice, c.NewPrice, c.Address)
			}
			if d.HasChanges() {
				os.Exit(exitChangesDetect)
			}
			return nil
		},
	}
}

func compareCmd() *cobra.Command {
	var out string
	cmd := &cobra.Command{
		Use:   "compare <city1> <city2>",
		Short: "Side-by-side report for two cities",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _ := config.Load()
			defaults := cfg.Defaults
			mkOpts := func(loc string) SearchOpts {
				return SearchOpts{
					Location: loc,
					MaxPrice: defaults.MaxPrice,
					MinBeds:  defaults.MinBeds,
					MinBaths: defaults.MinBaths,
					MinSqFt:  defaults.MinSqFt,
					Types:    defaults.Types,
					NoEnrich: false,
				}
			}
			fmt.Fprintln(os.Stderr, "Searching city 1 ...")
			a, err := runSearch(mkOpts(args[0]))
			if err != nil {
				return err
			}
			fmt.Fprintln(os.Stderr, "Searching city 2 ...")
			b, err := runSearch(mkOpts(args[1]))
			if err != nil {
				return err
			}
			if out == "" {
				out = "homelens-compare-" + slugify(args[0]) + "-vs-" + slugify(args[1]) + ".html"
			}
			f, err := os.Create(out)
			if err != nil {
				return err
			}
			defer f.Close()
			if err := compare.Render(
				compare.CitySet{Name: a.Location, Homes: a.Homes, Zips: a.Zips, Livability: a.Livability},
				compare.CitySet{Name: b.Location, Homes: b.Homes, Zips: b.Zips, Livability: b.Livability},
				"today", f); err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "✓ Wrote %s\n", out)
			fmt.Println(out)
			return nil
		},
	}
	cmd.Flags().StringVar(&out, "out", "", "output HTML path")
	return cmd
}

func listingCmd() *cobra.Command {
	var out string
	var noAmenities bool
	cmd := &cobra.Command{
		Use:   "listing <redfin-url>",
		Short: "Single-listing deep dive (census tract + OSM amenities + walkability)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			url := args[0]
			fmt.Fprintln(os.Stderr, "Fetching Redfin listing page ...")
			d, err := listing.Fetch(url)
			if err != nil {
				return err
			}

			var tract *census.GeocodeResult
			if d.Latitude != 0 && d.Longitude != 0 {
				fmt.Fprintln(os.Stderr, "Resolving census tract ...")
				if t, err := census.Geocode(d.Latitude, d.Longitude); err == nil {
					tract = t
				} else {
					fmt.Fprintln(os.Stderr, "  tract lookup failed:", err)
				}
			} else {
				fmt.Fprintln(os.Stderr, "  no lat/lng — skipping tract & amenities")
			}

			var amens *osm.Amenities
			if !noAmenities && d.Latitude != 0 && d.Longitude != 0 {
				fmt.Fprintln(os.Stderr, "Querying OSM Overpass for nearby amenities (1 mi) ...")
				if a, err := osm.Fetch(d.Latitude, d.Longitude, 1609); err == nil {
					amens = a
				} else {
					fmt.Fprintln(os.Stderr, "  OSM query failed:", err)
				}
			}

			if out == "" {
				out = "homelens-listing.html"
			}
			f, err := os.Create(out)
			if err != nil {
				return err
			}
			defer f.Close()
			if err := listing.RenderDeepDive(listing.DeepDive{
				URL: url, Detail: d, Tract: tract, Amenities: amens,
			}, f); err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "✓ Wrote %s\n", out)
			fmt.Println(out)
			return nil
		},
	}
	cmd.Flags().StringVar(&out, "out", "", "output HTML path")
	cmd.Flags().BoolVar(&noAmenities, "no-amenities", false, "skip OSM amenity lookup")
	return cmd
}

func reportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "report",
		Short: "[STUB] Re-render previous results — use `search` directly for now",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("report is a v0.3 feature — for now, re-run `homelens-pp-cli search` with --theme to re-render.")
			return nil
		},
	}
}

func shareCmd() *cobra.Command {
	var private bool
	cmd := &cobra.Command{
		Use:   "share <report.html>",
		Short: "Upload report as a GitHub Gist via gh CLI",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			url, err := share.Gist(args[0], !private)
			if err != nil {
				return err
			}
			fmt.Println(url)
			return nil
		},
	}
	cmd.Flags().BoolVar(&private, "private", false, "secret gist (not searchable)")
	return cmd
}

func profileCmd() *cobra.Command {
	c := &cobra.Command{Use: "profile", Short: "Manage filter profiles"}
	c.AddCommand(&cobra.Command{
		Use: "list", Short: "List available profiles",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			for name, p := range cfg.Profiles {
				marker := " "
				if name == cfg.ActiveProfile {
					marker = "*"
				}
				fmt.Printf(" %s %s — max=$%d min_beds=%d min_sqft=%d types=%v\n",
					marker, name, p.MaxPrice, p.MinBeds, p.MinSqFt, p.Types)
			}
			return nil
		},
	})
	c.AddCommand(&cobra.Command{
		Use: "use <name>", Short: "Set the active profile",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if _, ok := cfg.Profiles[args[0]]; !ok {
				fmt.Fprintf(os.Stderr, "no such profile: %s\n", args[0])
				os.Exit(exitUserError)
			}
			cfg.ActiveProfile = args[0]
			if err := config.Save(cfg); err != nil {
				return err
			}
			fmt.Printf("✓ Active profile: %s\n", args[0])
			return nil
		},
	})
	return c
}

func configCmd() *cobra.Command {
	c := &cobra.Command{Use: "config", Short: "Manage configuration"}
	c.AddCommand(&cobra.Command{
		Use: "show", Short: "Print current resolved config",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			b, _ := json.MarshalIndent(cfg, "", "  ")
			fmt.Println(string(b))
			fmt.Println()
			fmt.Println("Path:", config.ConfigPath())
			return nil
		},
	})
	c.AddCommand(&cobra.Command{
		Use: "edit", Short: "Print the config file path",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println(config.ConfigPath())
			return nil
		},
	})
	return c
}

func agentContextCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "agent-context",
		Short: "Emit a JSON manifest describing CLI capabilities for agents",
		RunE: func(cmd *cobra.Command, args []string) error {
			manifest := map[string]any{
				"name":        "homelens-pp-cli",
				"version":     version,
				"description": "Property search & neighborhood enrichment tool.",
				"exit_codes": map[string]int{
					"ok": exitOK, "user_error": exitUserError, "upstream_error": exitUpstreamError,
					"rate_limited": exitRateLimited, "auth_missing": exitAuthMissing,
					"no_results": exitNoResults, "changes_detected": exitChangesDetect,
				},
				"commands": []map[string]any{
					{"name": "search", "status": "working"},
					{"name": "save", "status": "working"},
					{"name": "list-searches", "status": "working"},
					{"name": "watch", "status": "working"},
					{"name": "compare", "status": "working"},
					{"name": "listing", "status": "working (basic — full deep-dive in v0.3)"},
					{"name": "share", "status": "working"},
					{"name": "init", "status": "working (interactive)"},
					{"name": "profile list/use", "status": "working"},
					{"name": "config show/edit", "status": "working"},
					{"name": "report", "status": "stub"},
				},
				"themes":          []string{"bloom", "modern", "classic", "minimal", "dark"},
				"output_formats":  []string{"html", "md", "json"},
				"stdout_contract": "search/compare print the output file path on stdout; all logging goes to stderr",
			}
			b, _ := json.MarshalIndent(manifest, "", "  ")
			fmt.Println(string(b))
			return nil
		},
	}
}

func doctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check that HomeLens dependencies and config are healthy",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			fmt.Println("HomeLens doctor")
			fmt.Println("───────────────")
			fmt.Printf("Config path:           %s\n", config.ConfigPath())
			if cfg.Census.APIKey != "" {
				fmt.Printf("Census API key:        ✓ (length %d)\n", len(cfg.Census.APIKey))
			} else {
				fmt.Println("Census API key:        ✗ missing")
			}
			fmt.Printf("Active profile:        %s\n", strDefault(cfg.ActiveProfile, "(none)"))
			fmt.Printf("Default theme:         %s\n", cfg.Defaults.Theme)
			fmt.Printf("Min sqft / max price:  %d / $%d\n", cfg.Defaults.MinSqFt, cfg.Defaults.MaxPrice)
			fmt.Printf("Profiles available:    %d\n", len(cfg.Profiles))
			return nil
		},
	}
}

func nz(v, fallback int) int {
	if v == 0 {
		return fallback
	}
	return v
}

func strDefault(s, d string) string {
	if s == "" {
		return d
	}
	return s
}
