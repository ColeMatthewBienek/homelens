// Package diff compares two snapshots of a search result set and surfaces
// new / removed / price-changed / status-changed listings.
package diff

import "github.com/ColeMatthewBienek/homelens/internal/redfin"

type Change struct {
	Kind     string // "new" | "removed" | "price_changed" | "status_changed"
	URL      string
	Address  string
	OldPrice int
	NewPrice int
	Home     redfin.Home
}

type Result struct {
	New      []Change
	Removed  []Change
	Changed  []Change
	Unchanged int
}

func Compute(prev, curr []redfin.Home) Result {
	pi := index(prev)
	ci := index(curr)
	var r Result
	for url, h := range ci {
		if old, ok := pi[url]; ok {
			if old.Price.Int() != h.Price.Int() {
				r.Changed = append(r.Changed, Change{
					Kind:     "price_changed",
					URL:      url,
					Address:  h.StreetLine.String(),
					OldPrice: old.Price.Int(),
					NewPrice: h.Price.Int(),
					Home:     h,
				})
			} else {
				r.Unchanged++
			}
		} else {
			r.New = append(r.New, Change{Kind: "new", URL: url, Address: h.StreetLine.String(), NewPrice: h.Price.Int(), Home: h})
		}
	}
	for url, h := range pi {
		if _, ok := ci[url]; !ok {
			r.Removed = append(r.Removed, Change{Kind: "removed", URL: url, Address: h.StreetLine.String(), OldPrice: h.Price.Int(), Home: h})
		}
	}
	return r
}

func (r Result) HasChanges() bool {
	return len(r.New) > 0 || len(r.Removed) > 0 || len(r.Changed) > 0
}

func index(homes []redfin.Home) map[string]redfin.Home {
	out := make(map[string]redfin.Home, len(homes))
	for _, h := range homes {
		out[h.URL] = h
	}
	return out
}
