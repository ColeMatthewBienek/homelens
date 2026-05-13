package main

import (
	"fmt"
	"os"
	"sort"
	"sync"

	"github.com/ColeMatthewBienek/homelens/internal/citydata"
	"github.com/ColeMatthewBienek/homelens/internal/config"
	"github.com/ColeMatthewBienek/homelens/internal/redfin"
	"github.com/ColeMatthewBienek/homelens/internal/score"
)

// SearchResult is the shared output shape used by `search`, `compare`, and `watch`.
type SearchResult struct {
	Location   string
	Slug       string
	Homes      []redfin.Home
	Zips       map[string]*citydata.ZipProfile
	ZipCity    map[string]string
	Livability map[string]int
	SortedZips []string
}

type SearchOpts struct {
	Location  string
	Slug      string
	MaxPrice  int
	MinBeds   int
	MinBaths  int
	MinSqFt   int
	Types     []string
	NoEnrich  bool
	Quiet     bool
}

func runSearch(opts SearchOpts) (*SearchResult, error) {
	slug := opts.Slug
	if slug == "" {
		if !opts.Quiet {
			fmt.Fprintf(os.Stderr, "Resolving Redfin slug for %q ...\n", opts.Location)
		}
		s, err := redfin.ResolveCity(opts.Location)
		if err != nil {
			return nil, fmt.Errorf("city resolution failed: %w", err)
		}
		slug = s
	}
	if !opts.Quiet {
		fmt.Fprintf(os.Stderr, "Fetching Redfin (%s) ...\n", slug)
	}
	homes, err := redfin.Search(slug, redfin.Filters{
		MaxPrice: opts.MaxPrice,
		MinBeds:  opts.MinBeds,
		MinBaths: opts.MinBaths,
		MinSqFt:  opts.MinSqFt,
		Types:    config.TypesToUIPT(opts.Types),
		Status:   1,
	}, 3)
	if err != nil {
		return nil, err
	}
	if !opts.Quiet {
		fmt.Fprintf(os.Stderr, "Got %d filtered listings.\n", len(homes))
	}

	res := &SearchResult{
		Location: opts.Location,
		Slug:     slug,
		Homes:    homes,
		Zips:     map[string]*citydata.ZipProfile{},
		ZipCity:  map[string]string{},
	}

	if !opts.NoEnrich && len(homes) > 0 {
		uniqueZips := map[string]bool{}
		for _, h := range homes {
			if h.Zip != "" {
				uniqueZips[h.Zip] = true
				if _, ok := res.ZipCity[h.Zip]; !ok {
					res.ZipCity[h.Zip] = h.City
				}
			}
		}
		if !opts.Quiet {
			fmt.Fprintf(os.Stderr, "Enriching %d unique ZIP codes ...\n", len(uniqueZips))
		}
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
					return
				}
				mu.Lock()
				res.Zips[z] = p
				mu.Unlock()
			}(z)
		}
		wg.Wait()
	}

	res.Livability = map[string]int{}
	for z := range res.Zips {
		res.Livability[z] = score.Livability(z, res.Zips)
	}
	res.SortedZips = make([]string, 0, len(res.Zips))
	for z := range res.Zips {
		res.SortedZips = append(res.SortedZips, z)
	}
	sort.Slice(res.SortedZips, func(i, j int) bool {
		return res.Livability[res.SortedZips[i]] > res.Livability[res.SortedZips[j]]
	})
	return res, nil
}
