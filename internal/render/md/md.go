// Package md renders search results as GitHub-flavored Markdown.
package md

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/ColeMatthewBienek/homelens/internal/citydata"
	"github.com/ColeMatthewBienek/homelens/internal/redfin"
)

type Data struct {
	Location   string
	Filters    Filters
	Homes      []redfin.Home
	Zips       map[string]*citydata.ZipProfile
	SortedZips []string
	Livability map[string]int
	ZipCity    map[string]string
}

type Filters struct {
	MinSqFt, MaxPrice, MinBeds, MinBaths int
	Types                                []string
}

func Render(d Data, w io.Writer) error {
	today := time.Now().Format("January 2, 2006")
	var b strings.Builder
	fmt.Fprintf(&b, "# %s — HomeLens\n\n", d.Location)
	fmt.Fprintf(&b, "*Live snapshot · %s · %d matches*\n\n", today, len(d.Homes))
	fmt.Fprintln(&b, "## Search criteria\n")
	fmt.Fprintln(&b, "| Filter | Value |")
	fmt.Fprintln(&b, "|---|---|")
	fmt.Fprintf(&b, "| Min sqft | %d |\n", d.Filters.MinSqFt)
	fmt.Fprintf(&b, "| Max price | $%s |\n", commaInt(d.Filters.MaxPrice))
	fmt.Fprintf(&b, "| Min beds / baths | %d / %d |\n", d.Filters.MinBeds, d.Filters.MinBaths)
	fmt.Fprintf(&b, "| Types | %s |\n\n", strings.Join(d.Filters.Types, ", "))

	if len(d.Zips) > 0 {
		fmt.Fprintln(&b, "## Neighborhoods (by Livability)\n")
		fmt.Fprintln(&b, "| ZIP | City | Income | BA+% | Poverty | Median home | Livability |")
		fmt.Fprintln(&b, "|---|---|---:|---:|---:|---:|---:|")
		for _, z := range d.SortedZips {
			zp := d.Zips[z]
			fmt.Fprintf(&b, "| %s | %s | $%s | %.1f%% | %.1f%% | $%s | **%d/100** |\n",
				z, d.ZipCity[z], commaInt(zp.MedianHouseholdIncome),
				zp.PctBachelorsOrHigher, zp.PovertyPct,
				commaInt(zp.MedianHouseValue), d.Livability[z])
		}
		fmt.Fprintln(&b)
	}

	fmt.Fprintln(&b, "## Listings\n")
	fmt.Fprintln(&b, "| # | Price | Type | Bd/Ba | Sqft | $/sqft | Year | ZIP | Liv | Address |")
	fmt.Fprintln(&b, "|---|---:|---|---:|---:|---:|---:|---:|---:|---|")
	for i, h := range d.Homes {
		liv := d.Livability[h.Zip]
		addr := h.StreetLine.String()
		if h.UnitNumber.Present() {
			addr += " #" + h.UnitNumber.String()
		}
		fmt.Fprintf(&b, "| %d | **$%s** | %s | %d/%g | %s | $%d | %d | %s | %d/100 | [%s](https://www.redfin.com%s) |\n",
			i+1, commaInt(h.Price.Int()), typeName(h.UIPropertyType),
			int(h.Beds), h.Baths, commaInt(h.SqFt.Int()), h.PricePerSqFt.Int(),
			h.YearBuilt.Int(), h.Zip, liv, addr+", "+h.Zip, h.URL)
	}
	fmt.Fprintf(&b, "\n*Generated %s by HomeLens.*\n", today)
	_, err := io.WriteString(w, b.String())
	return err
}

func commaInt(n int) string {
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
}

func typeName(t int) string {
	switch t {
	case 1:
		return "House"
	case 2:
		return "Condo"
	case 3:
		return "Townhouse"
	case 4:
		return "Multi"
	case 5:
		return "Land"
	}
	return "?"
}
