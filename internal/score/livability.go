// Package score computes the Livability composite — a within-search
// percentile rank across 4 equally-weighted metrics. It is a RELATIVE
// score against the other ZIPs in the same search, not a national score.
package score

import (
	"sort"

	"github.com/ColeMatthewBienek/homelens/internal/citydata"
)

type Metrics struct {
	Income       int
	Education    float64
	Poverty      float64
	HomeValue    int
}

// Livability returns 0-100 for one ZIP given the full set of ZIPs in the
// search. Equally weighted across income (↑), education (↑), poverty (↓),
// home value (↑).
func Livability(target string, zips map[string]*citydata.ZipProfile) int {
	if len(zips) < 2 {
		return 50
	}
	z := zips[target]
	if z == nil {
		return 0
	}
	inc := pctRank(target, zips, func(p *citydata.ZipProfile) float64 { return float64(p.MedianHouseholdIncome) }, false)
	edu := pctRank(target, zips, func(p *citydata.ZipProfile) float64 { return p.PctBachelorsOrHigher }, false)
	pov := pctRank(target, zips, func(p *citydata.ZipProfile) float64 { return p.PovertyPct }, true)
	val := pctRank(target, zips, func(p *citydata.ZipProfile) float64 { return float64(p.MedianHouseValue) }, false)
	return int((inc + edu + pov + val) / 4.0 + 0.5)
}

func pctRank(target string, zips map[string]*citydata.ZipProfile, get func(*citydata.ZipProfile) float64, inverted bool) float64 {
	vals := make([]float64, 0, len(zips))
	for _, z := range zips {
		vals = append(vals, get(z))
	}
	sort.Float64s(vals)
	v := get(zips[target])
	below := 0
	for _, x := range vals {
		if x < v {
			below++
		}
	}
	p := 100.0 * float64(below) / float64(len(vals)-1)
	if inverted {
		return 100 - p
	}
	return p
}

// GroupMedians returns the median across all ZIPs for each metric, used
// for the per-ZIP "why it scored this" explanations.
type GroupMedians struct {
	Income    float64
	Education float64
	Poverty   float64
	HomeValue float64
}

func ComputeGroupMedians(zips map[string]*citydata.ZipProfile) GroupMedians {
	inc := collect(zips, func(p *citydata.ZipProfile) float64 { return float64(p.MedianHouseholdIncome) })
	edu := collect(zips, func(p *citydata.ZipProfile) float64 { return p.PctBachelorsOrHigher })
	pov := collect(zips, func(p *citydata.ZipProfile) float64 { return p.PovertyPct })
	val := collect(zips, func(p *citydata.ZipProfile) float64 { return float64(p.MedianHouseValue) })
	return GroupMedians{med(inc), med(edu), med(pov), med(val)}
}

func collect(zips map[string]*citydata.ZipProfile, get func(*citydata.ZipProfile) float64) []float64 {
	out := make([]float64, 0, len(zips))
	for _, z := range zips {
		out = append(out, get(z))
	}
	return out
}

func med(vs []float64) float64 {
	if len(vs) == 0 {
		return 0
	}
	s := append([]float64(nil), vs...)
	sort.Float64s(s)
	m := len(s) / 2
	if len(s)%2 == 1 {
		return s[m]
	}
	return (s[m-1] + s[m]) / 2
}
