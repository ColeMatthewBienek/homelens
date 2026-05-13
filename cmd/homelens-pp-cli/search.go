package main

import (
	"fmt"
	"html/template"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/spf13/cobra"

	"github.com/ColeMatthewBienek/homelens/internal/citydata"
	"github.com/ColeMatthewBienek/homelens/internal/config"
	"github.com/ColeMatthewBienek/homelens/internal/mapview"
	htmlrender "github.com/ColeMatthewBienek/homelens/internal/render/html"
	mdrender "github.com/ColeMatthewBienek/homelens/internal/render/md"
	pdfrender "github.com/ColeMatthewBienek/homelens/internal/render/pdf"
	"github.com/ColeMatthewBienek/homelens/internal/redfin"
	"github.com/ColeMatthewBienek/homelens/internal/score"
	"github.com/ColeMatthewBienek/homelens/internal/store"
)

// Exit codes (typed per spec)
const (
	exitOK             = 0
	exitUserError      = 2
	exitUpstreamError  = 3
	exitRateLimited    = 4
	exitAuthMissing    = 5
	exitNoResults      = 7
	exitChangesDetect  = 9
)

func searchCmd() *cobra.Command {
	var (
		flagMinSqft   int
		flagMaxPrice  int
		flagMinBeds   int
		flagMinBaths  int
		flagTypes     string
		flagStatus    string
		flagSlug      string
		flagOut       string
		flagTheme     string
		flagProfile   string
		flagChunk     int
		flagPage      int
		flagAll       bool
		flagJSON      bool
		flagNoEnrich  bool
		flagMap       bool
		flagInlineMap bool
		flagMarkdown  bool
		flagPDF       bool
	)
	cmd := &cobra.Command{
		Use:   "search [city-state | saved-search-name]",
		Short: "Search for properties in a city (or run a saved search)",
		Long:  "Pull Redfin listings filtered by your config defaults, enrich with neighborhood demographics, and render an HTML report.",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			location := strings.Join(args, " ")

			cfg, err := config.Load()
			if err != nil {
				return err
			}
			defaults := config.ResolveProfile(cfg, flagProfile)

			// Try loading as saved search first
			if saved, err := store.LoadSearch(location); err == nil && saved != nil {
				location = saved.Location
				if flagSlug == "" {
					flagSlug = saved.Slug
				}
				if flagMaxPrice == 0 {
					flagMaxPrice = saved.MaxPrice
				}
				if flagMinBeds == 0 {
					flagMinBeds = saved.MinBeds
				}
				if flagMinBaths == 0 {
					flagMinBaths = saved.MinBaths
				}
				if flagMinSqft == 0 {
					flagMinSqft = saved.MinSqFt
				}
				if flagTypes == "" && len(saved.Types) > 0 {
					flagTypes = strings.Join(saved.Types, ",")
				}
				if flagTheme == "" {
					flagTheme = saved.Theme
				}
			}

			// Apply config defaults as fallback for unset flags
			if flagMinSqft == 0 {
				flagMinSqft = defaults.MinSqFt
			}
			if flagMaxPrice == 0 {
				flagMaxPrice = defaults.MaxPrice
			}
			if flagMinBeds == 0 {
				flagMinBeds = defaults.MinBeds
			}
			if flagMinBaths == 0 {
				flagMinBaths = defaults.MinBaths
			}
			var types []string
			if flagTypes != "" {
				types = strings.Split(flagTypes, ",")
			} else {
				types = defaults.Types
			}
			if flagTheme == "" {
				flagTheme = defaults.Theme
			}
			if flagChunk == 0 {
				flagChunk = defaults.Chunk
				if flagChunk == 0 {
					flagChunk = 25
				}
			}

			// Resolve city → Redfin slug if not provided
			slug := flagSlug
			if slug == "" {
				fmt.Fprintf(os.Stderr, "Resolving Redfin slug for %q ...\n", location)
				s, err := redfin.ResolveCity(location)
				if err != nil {
					fmt.Fprintf(os.Stderr, "city resolution failed: %v\n", err)
					fmt.Fprintln(os.Stderr, "Tip: pass --slug city/<ID>/<ST>/<Name> directly if you know it.")
					os.Exit(exitUserError)
				}
				slug = s
				fmt.Fprintf(os.Stderr, "Slug: %s\n", slug)
			}

			fmt.Fprintf(os.Stderr, "Fetching Redfin (%s) ...\n", slug)
			homes, err := redfin.Search(slug, redfin.Filters{
				MaxPrice: flagMaxPrice,
				MinBeds:  flagMinBeds,
				MinBaths: flagMinBaths,
				MinSqFt:  flagMinSqft,
				Types:    config.TypesToUIPT(types),
				Status:   1,
			}, 3)
			if err != nil {
				fmt.Fprintln(os.Stderr, "redfin error:", err)
				os.Exit(exitUpstreamError)
			}
			fmt.Fprintf(os.Stderr, "Got %d filtered listings.\n", len(homes))

			if len(homes) == 0 {
				fmt.Fprintln(os.Stderr, "No listings matched. Try widening the filters.")
				os.Exit(exitNoResults)
			}

			// Pagination
			if !flagAll {
				start := (flagPage - 1) * flagChunk
				if start < 0 {
					start = 0
				}
				if start >= len(homes) {
					start = 0
				}
				end := start + flagChunk
				if end > len(homes) {
					end = len(homes)
				}
				homes = homes[start:end]
			}

			// Enrich with city-data ZIP profiles (one fetch per unique ZIP)
			zips := map[string]*citydata.ZipProfile{}
			zipCity := map[string]string{}
			if !flagNoEnrich {
				uniqueZips := map[string]bool{}
				for _, h := range homes {
					if h.Zip != "" {
						uniqueZips[h.Zip] = true
						if _, ok := zipCity[h.Zip]; !ok {
							zipCity[h.Zip] = h.City
						}
					}
				}
				fmt.Fprintf(os.Stderr, "Enriching %d unique ZIP codes from city-data.com ...\n", len(uniqueZips))
				var mu sync.Mutex
				var wg sync.WaitGroup
				sem := make(chan struct{}, 4)
				for z := range uniqueZips {
					wg.Add(1)
					sem <- struct{}{}
					go func(z string) {
						defer wg.Done()
						defer func() { <-sem }()
						p, err := citydata.FetchZip(z)
						if err != nil {
							fmt.Fprintf(os.Stderr, "  ZIP %s enrichment failed: %v\n", z, err)
							return
						}
						mu.Lock()
						zips[z] = p
						mu.Unlock()
					}(z)
				}
				wg.Wait()
			}

			// Livability scores
			livability := map[string]int{}
			for z := range zips {
				livability[z] = score.Livability(z, zips)
			}

			// Sort ZIPs by livability desc
			sortedZips := make([]string, 0, len(zips))
			for z := range zips {
				sortedZips = append(sortedZips, z)
			}
			sort.Slice(sortedZips, func(i, j int) bool {
				return livability[sortedZips[i]] > livability[sortedZips[j]]
			})

			// Render
			ext := ".html"
			if flagMarkdown {
				ext = ".md"
			}
			out := flagOut
			if out == "" {
				out = "homelens-" + slugify(location) + ext
			}

			f, err := os.Create(out)
			if err != nil {
				return err
			}
			defer f.Close()

			if flagMarkdown {
				err = mdrender.Render(mdrender.Data{
					Location: location,
					Filters: mdrender.Filters{
						MinSqFt: flagMinSqft, MaxPrice: flagMaxPrice,
						MinBeds: flagMinBeds, MinBaths: flagMinBaths, Types: types,
					},
					Homes: homes, Zips: zips, SortedZips: sortedZips,
					ZipCity: zipCity, Livability: livability,
				}, f)
			} else {
				var mapHTML template.HTML
				if flagMap || flagInlineMap {
					var mh template.HTML
					var mErr error
					if flagInlineMap {
						mh, mErr = mapview.BuildInline(homes, livability)
					} else {
						mh, mErr = mapview.Build(homes, livability)
					}
					if mErr == nil {
						mapHTML = mh
					}
				}
				err = htmlrender.Render(flagTheme, htmlrender.Data{
					Location: location,
					Filters: htmlrender.FiltersView{
						MinSqFt: flagMinSqft, MaxPrice: flagMaxPrice,
						MinBeds: flagMinBeds, MinBaths: flagMinBaths, Types: types,
					},
					Homes: homes, Zips: zips, SortedZips: sortedZips,
					ZipCity: zipCity, Livability: livability, MapHTML: mapHTML,
				}, f)
			}
			if err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "✓ Wrote %s\n", out)

			if flagPDF && !flagMarkdown {
				pdfPath := strings.TrimSuffix(out, ".html") + ".pdf"
				fmt.Fprintf(os.Stderr, "Rendering PDF via headless Chrome ...\n")
				if err := pdfrender.FromHTMLFile(out, pdfPath); err != nil {
					fmt.Fprintf(os.Stderr, "PDF render failed: %v\n", err)
				} else {
					fmt.Fprintf(os.Stderr, "✓ Wrote %s\n", pdfPath)
					fmt.Println(pdfPath)
					return nil
				}
			}
			fmt.Println(out) // stdout = the path, for agents to consume
			return nil
		},
	}
	cmd.Flags().IntVar(&flagMinSqft, "min-sqft", 0, "minimum square feet")
	cmd.Flags().IntVar(&flagMaxPrice, "max-price", 0, "maximum price")
	cmd.Flags().IntVar(&flagMinBeds, "min-beds", 0, "minimum bedrooms")
	cmd.Flags().IntVar(&flagMinBaths, "min-baths", 0, "minimum bathrooms")
	cmd.Flags().StringVar(&flagTypes, "types", "", "comma-separated: house,condo,townhouse,multi,land")
	cmd.Flags().StringVar(&flagStatus, "status", "for-sale", "for-sale | sold | pending")
	cmd.Flags().StringVar(&flagSlug, "slug", "", "Redfin region slug to skip city resolution")
	cmd.Flags().StringVar(&flagOut, "out", "", "output HTML path (default: homelens-<city>.html)")
	cmd.Flags().StringVar(&flagTheme, "theme", "", "theme: bloom (other themes stubbed for v0)")
	cmd.Flags().StringVar(&flagProfile, "profile", "", "filter profile name to apply")
	cmd.Flags().IntVar(&flagChunk, "chunk", 0, "results per page (default 25)")
	cmd.Flags().IntVar(&flagPage, "page", 1, "page number (1-indexed)")
	cmd.Flags().BoolVar(&flagAll, "all", false, "return all results, no pagination")
	cmd.Flags().BoolVar(&flagJSON, "json", false, "emit results as JSON to stdout (no HTML)")
	cmd.Flags().BoolVar(&flagNoEnrich, "no-enrich", false, "skip city-data enrichment (faster, no livability scores)")
	cmd.Flags().BoolVar(&flagMap, "map", false, "embed Leaflet map (loaded from unpkg CDN)")
	cmd.Flags().BoolVar(&flagInlineMap, "inline-map", false, "embed Leaflet map with inlined JS+CSS (fully offline, +160KB)")
	cmd.Flags().BoolVar(&flagMarkdown, "md", false, "emit Markdown instead of HTML")
	cmd.Flags().BoolVar(&flagPDF, "pdf", false, "render output to PDF via headless Chrome (requires Chrome/Edge installed)")
	return cmd
}

func slugify(s string) string {
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, ",", "")
	s = strings.ReplaceAll(s, " ", "-")
	return s
}
