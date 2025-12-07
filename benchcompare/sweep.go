package benchcompare

import (
	"fmt"
	"strconv"
	"strings"
)

// DefaultFPRSweepRates are the false positive targets used when -sweep-fpr is set
// without custom -p-values.
var DefaultFPRSweepRates = []float64{0.001, 0.01, 0.1}

// CompareFPRSweep runs ScenarioAdd at each rate and returns one comparison per
// target. Hash sets are unchanged; only Bloom filter sizing varies with p.
func CompareFPRSweep(cfg Config, rates []float64) ([]Comparison, error) {
	if len(rates) == 0 {
		return nil, fmt.Errorf("benchcompare: FPR sweep requires at least one rate")
	}
	out := make([]Comparison, 0, len(rates))
	for _, p := range rates {
		if p <= 0 || p >= 1 {
			return nil, fmt.Errorf("benchcompare: FPR sweep rate %v must be in (0, 1)", p)
		}
		sweepCfg := cfg
		sweepCfg.FalsePositiveRate = p
		cmp, err := compareAdd(sweepCfg)
		if err != nil {
			return nil, err
		}
		out = append(out, cmp)
	}
	return out, nil
}

// ParseFPRRates parses comma-separated false positive rates.
func ParseFPRRates(raw string) ([]float64, error) {
	parts := strings.Split(raw, ",")
	rates := make([]float64, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		p, err := strconv.ParseFloat(part, 64)
		if err != nil {
			return nil, fmt.Errorf("benchcompare: invalid FPR rate %q: %w", part, err)
		}
		rates = append(rates, p)
	}
	if len(rates) == 0 {
		return nil, fmt.Errorf("benchcompare: no FPR rates in %q", raw)
	}
	return rates, nil
}
