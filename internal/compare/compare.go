// Package compare produces a side-by-side HTML report for two cities.
package compare

import (
	"fmt"
	"html/template"
	"io"
	"sort"

	"github.com/ColeMatthewBienek/homelens/internal/citydata"
	"github.com/ColeMatthewBienek/homelens/internal/redfin"
)

type CitySet struct {
	Name       string
	Homes      []redfin.Home
	Zips       map[string]*citydata.ZipProfile
	Livability map[string]int
}

const tmpl = `<!DOCTYPE html><html><head><meta charset="UTF-8"><title>{{.A.Name}} vs {{.B.Name}} — HomeLens compare</title>
<style>
body{font-family:'Inter',system-ui,sans-serif;background:#f7fafc;color:#1a202c;margin:0;line-height:1.6}
.wrap{max-width:1280px;margin:0 auto;padding:2rem}
h1{font-size:2.2rem;margin-bottom:.5rem;letter-spacing:-.02em}
.cols{display:grid;grid-template-columns:1fr 1fr;gap:1.5rem;margin-top:2rem}
.col{background:#fff;border:1px solid #e2e8f0;border-radius:12px;padding:1.5rem}
.col h2{margin-bottom:1rem;color:#1a365d;font-size:1.4rem}
.stat{display:flex;justify-content:space-between;padding:.5rem 0;border-bottom:1px solid #f0f4f8;font-size:.95rem}
.stat strong{color:#1a365d}
.l-row{display:flex;justify-content:space-between;padding:.4rem 0;border-bottom:1px solid #f0f4f8;font-size:.85rem}
.l-row a{color:#2c5282;text-decoration:none}
.win{background:#e6f7e6;border-left:3px solid #0d8a4a;padding-left:.5rem}
.lose{background:#fef5e7;border-left:3px solid #c79822;padding-left:.5rem}
.summary-bar{background:#1a365d;color:#fff;padding:1rem 1.5rem;border-radius:12px;margin-bottom:2rem;display:flex;justify-content:space-around}
.summary-bar div{text-align:center}
.summary-bar .v{font-size:1.8rem;font-weight:800;color:#d69e2e}
.summary-bar .l{font-size:.75rem;text-transform:uppercase;letter-spacing:.1em;opacity:.85}
</style></head><body><div class="wrap">
<h1>{{.A.Name}} <span style="color:#666">vs</span> {{.B.Name}}</h1>
<p style="color:#666;margin-bottom:2rem">Side-by-side comparison · matched filter set · {{.Today}}</p>
<div class="summary-bar">
<div><div class="v">{{len .A.Homes}} vs {{len .B.Homes}}</div><div class="l">Listings</div></div>
<div><div class="v">${{fmtK .AStats.MedPrice}} vs ${{fmtK .BStats.MedPrice}}</div><div class="l">Median price</div></div>
<div><div class="v">{{.AStats.MedSqFt}} vs {{.BStats.MedSqFt}}</div><div class="l">Median sqft</div></div>
</div>
<div class="cols">
{{template "cityCol" dict "C" .A "Stats" .AStats}}
{{template "cityCol" dict "C" .B "Stats" .BStats}}
</div></div></body></html>
{{define "cityCol"}}
<div class="col">
<h2>{{.C.Name}}</h2>
<div class="stat"><span>Matches</span><strong>{{len .C.Homes}}</strong></div>
<div class="stat"><span>Min price</span><strong>${{fmtK .Stats.MinPrice}}</strong></div>
<div class="stat"><span>Median price</span><strong>${{fmtK .Stats.MedPrice}}</strong></div>
<div class="stat"><span>Max price</span><strong>${{fmtK .Stats.MaxPrice}}</strong></div>
<div class="stat"><span>Median sqft</span><strong>{{.Stats.MedSqFt}}</strong></div>
<div class="stat"><span>Unique ZIPs</span><strong>{{len .C.Zips}}</strong></div>
<h2 style="font-size:1rem;margin-top:1.5rem;margin-bottom:.75rem">Top 5 by price</h2>
{{range $i,$h := topN .C.Homes 5}}
<div class="l-row"><a href="https://www.redfin.com{{$h.URL}}" target="_blank">{{$h.StreetLine.String}} · {{$h.Zip}}</a><strong>${{fmtK $h.Price.Int}}</strong></div>
{{end}}
</div>
{{end}}
`

type stats struct {
	MinPrice, MaxPrice, MedPrice, MedSqFt int
}

type viewData struct {
	A, B   CitySet
	AStats stats
	BStats stats
	Today  string
}

func computeStats(homes []redfin.Home) stats {
	if len(homes) == 0 {
		return stats{}
	}
	prices := make([]int, len(homes))
	sqfts := make([]int, len(homes))
	for i, h := range homes {
		prices[i] = h.Price.Int()
		sqfts[i] = h.SqFt.Int()
	}
	sort.Ints(prices)
	sort.Ints(sqfts)
	var ps, qs int
	for _, p := range prices {
		ps += p
	}
	for _, q := range sqfts {
		qs += q
	}
	return stats{
		MinPrice: prices[0],
		MaxPrice: prices[len(prices)-1],
		MedPrice: ps / len(prices),
		MedSqFt:  qs / len(sqfts),
	}
}

func Render(a, b CitySet, today string, w io.Writer) error {
	funcs := template.FuncMap{
		"fmtK": func(n int) string {
			if n >= 1000 {
				return fmt.Sprintf("%dK", n/1000)
			}
			return fmt.Sprintf("%d", n)
		},
		"topN": func(homes []redfin.Home, n int) []redfin.Home {
			if n > len(homes) {
				n = len(homes)
			}
			return homes[:n]
		},
		"dict": func(values ...any) map[string]any {
			m := map[string]any{}
			for i := 0; i+1 < len(values); i += 2 {
				m[values[i].(string)] = values[i+1]
			}
			return m
		},
	}
	t, err := template.New("compare").Funcs(funcs).Parse(tmpl)
	if err != nil {
		return err
	}
	return t.Execute(w, viewData{
		A: a, B: b,
		AStats: computeStats(a.Homes),
		BStats: computeStats(b.Homes),
		Today:  today,
	})
}
