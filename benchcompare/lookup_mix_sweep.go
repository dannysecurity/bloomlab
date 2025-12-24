package benchcompare

import (
	"fmt"
	"strconv"
	"strings"
)

// DefaultLookupMixRatios are hit fractions compared when -sweep-mix is set without
// custom -mix-values.
var DefaultLookupMixRatios = []float64{0, 0.25, 0.5, 0.75, 1.0}

// CompareLookupMixSweep runs ScenarioContainsMixed at each hit ratio and returns
// one comparison per ratio. Bloom filter and hash set both probe the same
// interleaved key stream; only the hit/miss mix changes.
func CompareLookupMixSweep(cfg Config, ratios []float64) ([]Comparison, error) {
	if len(ratios) == 0 {
		return nil, fmt.Errorf("benchcompare: lookup mix sweep requires at least one ratio")
	}
	out := make([]Comparison, 0, len(ratios))
	for _, ratio := range ratios {
		if ratio < 0 || ratio > 1 {
			return nil, fmt.Errorf("benchcompare: lookup mix ratio %v must be in [0, 1]", ratio)
		}
		sweepCfg := cfg
		sweepCfg.LookupHitRatio = ratio
		cmp, err := compareContainsMixed(sweepCfg)
		if err != nil {
			return nil, err
		}
		out = append(out, cmp)
	}
	return out, nil
}

// ParseLookupMixRatios parses comma-separated lookup hit ratios.
func ParseLookupMixRatios(raw string) ([]float64, error) {
	parts := strings.Split(raw, ",")
	ratios := make([]float64, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		r, err := strconv.ParseFloat(part, 64)
		if err != nil {
			return nil, fmt.Errorf("benchcompare: invalid lookup mix ratio %q: %w", part, err)
		}
		ratios = append(ratios, r)
	}
	if len(ratios) == 0 {
		return nil, fmt.Errorf("benchcompare: no lookup mix ratios in %q", raw)
	}
	return ratios, nil
}
