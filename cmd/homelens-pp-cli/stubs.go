package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ColeMatthewBienek/homelens/internal/config"
	"github.com/ColeMatthewBienek/homelens/internal/store"
)

// Stubs for the remaining features. Each prints a clear TODO message and
// exits 0 (informational). The full implementation will land in subsequent
// sessions — see the README "Roadmap" section.

func initCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Interactive bootstrap — writes ~/.config/homelens/config.toml",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			fmt.Println("HomeLens init")
			fmt.Println("─────────────")
			fmt.Printf("Config will be written to: %s\n\n", config.ConfigPath())
			fmt.Println("v0 ships with sane defaults — full interactive wizard is on the roadmap.")
			fmt.Printf("Current defaults: min_sqft=%d, max_price=%d, min_beds=%d, min_baths=%d, types=%v, theme=%s\n",
				cfg.Defaults.MinSqFt, cfg.Defaults.MaxPrice, cfg.Defaults.MinBeds, cfg.Defaults.MinBaths,
				cfg.Defaults.Types, cfg.Defaults.Theme)
			if cfg.Census.APIKey != "" {
				fmt.Println("Census API key: ✓ detected (from census-pp-cli config)")
			} else {
				fmt.Println("Census API key: ✗ not set (sign up free at https://api.census.gov/data/key_signup.html)")
			}
			fmt.Println()
			fmt.Println("Writing config with current defaults to disk so you can edit it.")
			return config.Save(cfg)
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
				Name:     name,
				Location: loc,
				Slug:     slug,
				MaxPrice: maxPrice,
				MinBeds:  minBeds,
				MinBaths: minBaths,
				MinSqFt:  minSqft,
				Status:   "for-sale",
				Theme:    theme,
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
		Short: "[STUB] Re-run a saved search and diff against last run",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("watch is stubbed in v0 — coming in next session.")
			fmt.Println("Workaround: run `homelens-pp-cli search <name>` repeatedly and compare manually.")
			return nil
		},
	}
}

func compareCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "compare <city1> <city2>",
		Short: "[STUB] Side-by-side report for two cities",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("compare is stubbed in v0. Workaround: run `homelens-pp-cli search` for each city separately.")
			return nil
		},
	}
}

func listingCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "listing <redfin-url-or-id>",
		Short: "[STUB] Single-listing deep dive",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("listing deep-dive is stubbed in v0. The search HTML already shows full per-listing info.")
			return nil
		},
	}
}

func reportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "report",
		Short: "[STUB] Re-render previous results into a new format/theme",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("report is stubbed. v0 always re-renders during `search`. PDF/MD outputs land next session.")
			return nil
		},
	}
}

func shareCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "share <report.html>",
		Short: "[STUB] Upload report as a public Gist via gh CLI",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("share is stubbed. v0 workaround:")
			fmt.Printf("  gh gist create --public %s\n", args[0])
			return nil
		},
	}
}

func profileCmd() *cobra.Command {
	c := &cobra.Command{Use: "profile", Short: "Manage filter profiles"}
	c.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List available profiles",
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
		Use:   "use <name>",
		Short: "Set the active profile",
		Args:  cobra.ExactArgs(1),
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
		Use:   "show",
		Short: "Print current resolved config",
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
		Use:   "edit",
		Short: "Print the config file path (open it in your editor)",
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
				"description": "Property search & neighborhood enrichment tool. Pulls Redfin, enriches with US Census + city-data.com, renders HTML reports.",
				"exit_codes": map[string]int{
					"ok":               exitOK,
					"user_error":       exitUserError,
					"upstream_error":   exitUpstreamError,
					"rate_limited":     exitRateLimited,
					"auth_missing":     exitAuthMissing,
					"no_results":       exitNoResults,
					"changes_detected": exitChangesDetect,
				},
				"commands": []map[string]any{
					{"name": "search", "status": "working", "example": "homelens-pp-cli search \"Vancouver, WA\" --max-price 650000 --min-sqft 1750"},
					{"name": "save", "status": "working", "example": "homelens-pp-cli save my-search \"Vancouver, WA\" --max-price 650000"},
					{"name": "list-searches", "status": "working"},
					{"name": "profile list/use", "status": "working"},
					{"name": "config show/edit", "status": "working"},
					{"name": "init", "status": "minimal — writes current defaults to config file"},
					{"name": "watch", "status": "stub — v0.2"},
					{"name": "compare", "status": "stub — v0.2"},
					{"name": "listing", "status": "stub — v0.2"},
					{"name": "report", "status": "stub — v0.2 (PDF, markdown)"},
					{"name": "share", "status": "stub — v0.2 (gh gist wrapper)"},
				},
				"themes":      []string{"maia"},
				"themes_planned": []string{"modern", "classic", "minimal", "dark"},
				"stdout_contract": "search prints the output file path on stdout (one line); all logging goes to stderr",
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
				fmt.Println("Census API key:        ✗ missing (tract-level enrichment will be skipped)")
			}
			fmt.Printf("Active profile:        %s\n", strDefault(cfg.ActiveProfile, "(none)"))
			fmt.Printf("Default theme:         %s\n", cfg.Defaults.Theme)
			fmt.Printf("Min sqft / max price:  %d / $%d\n", cfg.Defaults.MinSqFt, cfg.Defaults.MaxPrice)
			fmt.Printf("Profiles available:    %d\n", len(cfg.Profiles))
			return nil
		},
	}
}

func strDefault(s, d string) string {
	if s == "" {
		return d
	}
	return s
}
