// Package html renders search results into a self-contained HTML document.
// Each theme is a Go text/template in themes/<theme>.tmpl. All themes share
// the same data shape so the same result set can be re-rendered in any theme.
package html

import (
	"embed"
	"fmt"
	"html/template"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/ColeMatthewBienek/homelens/internal/citydata"
	"github.com/ColeMatthewBienek/homelens/internal/redfin"
)

//go:embed themes/*.tmpl
var themesFS embed.FS

type Stats struct {
	MinPrice int
	MaxPrice int
	MedPrice int
	MedSqFt  int
}

type Data struct {
	Location   string
	Today      string
	Filters    FiltersView
	Homes      []redfin.Home
	Zips       map[string]*citydata.ZipProfile
	SortedZips []string
	ZipCity    map[string]string
	Livability map[string]int
	Stats      Stats
	Count      int
}

type FiltersView struct {
	MinSqFt  int
	MaxPrice int
	MinBeds  int
	MinBaths int
	Types    []string
}

func Render(theme string, d Data, w io.Writer) error {
	d.Today = time.Now().Format("January 2, 2006")
	d.Count = len(d.Homes)
	d.Stats = computeStats(d.Homes)

	name := "themes/" + theme + ".tmpl"
	tmpl, err := template.New(theme + ".tmpl").Funcs(funcMap()).ParseFS(themesFS, name)
	if err != nil {
		// Fallback to maia if theme not found
		tmpl, err = template.New("maia.tmpl").Funcs(funcMap()).ParseFS(themesFS, "themes/maia.tmpl")
		if err != nil {
			return err
		}
	}
	return tmpl.Execute(w, d)
}

func computeStats(homes []redfin.Home) Stats {
	var s Stats
	if len(homes) == 0 {
		return s
	}
	prices := make([]int, len(homes))
	sqfts := make([]int, len(homes))
	for i, h := range homes {
		prices[i] = h.Price.Int()
		sqfts[i] = h.SqFt.Int()
	}
	s.MinPrice = prices[0]
	s.MaxPrice = prices[0]
	priceSum := 0
	sqftSum := 0
	for _, p := range prices {
		if p < s.MinPrice {
			s.MinPrice = p
		}
		if p > s.MaxPrice {
			s.MaxPrice = p
		}
		priceSum += p
	}
	for _, q := range sqfts {
		sqftSum += q
	}
	s.MedPrice = priceSum / len(prices)
	s.MedSqFt = sqftSum / len(sqfts)
	return s
}

func funcMap() template.FuncMap {
	return template.FuncMap{
		"fmtK": func(n int) string {
			if n >= 1000 {
				return fmt.Sprintf("%dK", n/1000)
			}
			return fmt.Sprintf("%d", n)
		},
		"fmtInt": func(n int) string {
			s := fmt.Sprintf("%d", n)
			if len(s) <= 3 {
				return s
			}
			var out []byte
			for i, c := range s {
				if i > 0 && (len(s)-i)%3 == 0 {
					out = append(out, ',')
				}
				out = append(out, byte(c))
			}
			return string(out)
		},
		"intf": func(f float64) string {
			return fmt.Sprintf("%d", int(f))
		},
		"floatf": func(f float64) string {
			if f == float64(int(f)) {
				return fmt.Sprintf("%d", int(f))
			}
			return fmt.Sprintf("%.1f", f)
		},
		"typeName": func(t int) string {
			switch t {
			case 1:
				return "House"
			case 2:
				return "Condo"
			case 3:
				return "Townhouse"
			case 4:
				return "Multi-family"
			case 5:
				return "Land"
			}
			return "Property"
		},
		"typesList": func(types []string) string {
			caps := make([]string, len(types))
			for i, t := range types {
				if len(t) > 0 {
					caps[i] = strings.ToUpper(t[:1]) + t[1:]
				}
			}
			return strings.Join(caps, " · ")
		},
		"unitSuffix": func(h redfin.Home) string {
			if h.UnitNumber.Present() && h.UnitNumber.String() != "" {
				return " #" + h.UnitNumber.String()
			}
			return ""
		},
		"isNew": func(h redfin.Home) bool {
			for _, s := range h.Sashes {
				if s.SashTypeName == "New" {
					return true
				}
			}
			return false
		},
		"hoaStr": func(h redfin.Home) string {
			if !h.HOA.Present() {
				return ""
			}
			n := h.HOA.Int()
			if n <= 0 {
				return ""
			}
			return fmt.Sprintf("$%d/mo", n)
		},
		"mapsQuery": func(h redfin.Home) string {
			full := fmt.Sprintf("%s, %s, %s %s", h.StreetLine.String(), h.City, h.State, h.Zip)
			return url.QueryEscape(full)
		},
		"scoreColor": func(s int) string {
			switch {
			case s >= 80:
				return "#7ab87a"
			case s >= 60:
				return "#a3c87a"
			case s >= 40:
				return "#e8c87a"
			case s >= 20:
				return "#e89a7a"
			}
			return "#d97070"
		},
		"scoreLabel": func(s int) string {
			switch {
			case s >= 80:
				return "Lovely neighborhood — top tier in this search"
			case s >= 60:
				return "Great spot — above average across the board"
			case s >= 40:
				return "Quiet & solid — typical for this search"
			case s >= 20:
				return "More affordable — trails the median here"
			}
			return "Lowest-cost area in your search"
		},
		"inc": func(i int) int { return i + 1 },
	}
}
