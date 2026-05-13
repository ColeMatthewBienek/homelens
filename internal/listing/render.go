package listing

import (
	"fmt"
	"html/template"
	"io"
	"time"

	"github.com/ColeMatthewBienek/homelens/internal/census"
	"github.com/ColeMatthewBienek/homelens/internal/osm"
)

type DeepDive struct {
	URL         string
	Detail      *Detail
	Tract       *census.GeocodeResult
	Amenities   *osm.Amenities
}

const tmpl = `<!DOCTYPE html>
<html><head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>{{.URL}} — HomeLens listing</title>
<style>
body{font-family:'Inter',system-ui,sans-serif;background:#f7fafc;color:#1a202c;margin:0;line-height:1.65}
.wrap{max-width:900px;margin:0 auto;padding:2rem 1.5rem}
h1{font-size:1.8rem;margin-bottom:.25rem;letter-spacing:-.02em}
.sub{color:#666;margin-bottom:2rem}
section{background:#fff;border:1px solid #e2e8f0;border-radius:12px;padding:1.5rem 1.75rem;margin-bottom:1.5rem;box-shadow:0 1px 3px rgba(0,0,0,.04)}
h2{font-size:1.1rem;color:#1a365d;margin-bottom:1rem;text-transform:uppercase;letter-spacing:.05em}
.kv{display:grid;grid-template-columns:160px 1fr;gap:.5rem 1rem;font-size:.95rem}
.kv dt{color:#666}
.amen-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(140px,1fr));gap:.85rem}
.amen{background:#f8fafc;padding:1rem;border-radius:8px;text-align:center}
.amen .v{font-size:1.6rem;font-weight:700;color:#1a365d}
.amen .l{color:#666;font-size:.78rem;text-transform:uppercase;letter-spacing:.05em;margin-top:.25rem}
.walk{display:flex;align-items:center;gap:1rem;margin-bottom:1rem}
.walk .score{font-size:3rem;font-weight:800;color:#1a365d}
.walk .label{color:#666;font-size:.95rem}
.cta{display:inline-block;background:#1a365d;color:#fff;padding:.7rem 1.4rem;border-radius:8px;text-decoration:none;font-weight:600}
.desc{font-size:.95rem;line-height:1.7;color:#333}
</style></head><body><div class="wrap">
<h1>Listing deep-dive</h1>
<div class="sub">{{.URL}}</div>

<section>
<h2>Property</h2>
<dl class="kv">
{{if .Detail.YearBuilt}}<dt>Year built</dt><dd>{{.Detail.YearBuilt}}</dd>{{end}}
{{if .Detail.LotSize}}<dt>Lot size</dt><dd>{{.Detail.LotSize}} sqft</dd>{{end}}
{{if .Tract}}<dt>Census tract</dt><dd>{{.Tract.TractID}} (FIPS {{.Tract.State}}{{.Tract.County}}{{.Tract.Tract}})</dd>{{end}}
</dl>
{{if .Detail.Description}}
<p style="margin-top:1.25rem"><strong>Description</strong></p>
<p class="desc">{{.Detail.Description}}</p>
{{end}}
</section>

{{if .Amenities}}
<section>
<h2>Walkability & amenities (1 mi)</h2>
<div class="walk">
<div class="score">{{.Amenities.Score}}</div>
<div class="label">
<strong>OSM walkability score</strong> · 0-100 composite of nearby amenity density<br>
Approximates Walk Score; computed from OpenStreetMap Overpass within 1 mile.
</div>
</div>
<div class="amen-grid">
<div class="amen"><div class="v">{{.Amenities.Restaurants}}</div><div class="l">Restaurants</div></div>
<div class="amen"><div class="v">{{.Amenities.Cafes}}</div><div class="l">Cafes</div></div>
<div class="amen"><div class="v">{{.Amenities.Grocery}}</div><div class="l">Grocery</div></div>
<div class="amen"><div class="v">{{.Amenities.Schools}}</div><div class="l">Schools</div></div>
<div class="amen"><div class="v">{{.Amenities.Parks}}</div><div class="l">Parks</div></div>
<div class="amen"><div class="v">{{.Amenities.Transit}}</div><div class="l">Transit stops</div></div>
<div class="amen"><div class="v">{{.Amenities.Pharmacies}}</div><div class="l">Pharmacies</div></div>
<div class="amen"><div class="v">{{.Amenities.Hospitals}}</div><div class="l">Hospitals</div></div>
<div class="amen"><div class="v">{{.Amenities.Gyms}}</div><div class="l">Gyms</div></div>
</div>
</section>
{{end}}

<section style="text-align:center">
<a class="cta" href="https://www.redfin.com{{.URL}}" target="_blank">View full listing on Redfin →</a>
</section>

<p style="text-align:center;color:#999;font-size:.85rem;margin-top:2rem">Generated {{.Today}} by HomeLens · Redfin · US Census · OpenStreetMap</p>
</div></body></html>`

func RenderDeepDive(d DeepDive, w io.Writer) error {
	t, err := template.New("listing").Parse(tmpl)
	if err != nil {
		return err
	}
	type view struct {
		DeepDive
		Today string
	}
	return t.Execute(w, view{DeepDive: d, Today: fmt.Sprintf("%s", time.Now().Format("January 2, 2006"))})
}
