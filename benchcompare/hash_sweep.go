package benchcompare

import (
	"fmt"
	"strings"

	"github.com/dannysecurity/bloomlab/bloom"
)

// DefaultHashSweepStrategies are the hash families compared when -sweep-hash is
// set without custom -hash-values.
var DefaultHashSweepStrategies = []bloom.Strategy{
	bloom.HashFNV,
	bloom.HashMurmur3,
	bloom.HashXXHash,
	bloom.HashWyhash,
	bloom.HashHighway,
	bloom.HashSipHash,
}

// CompareHashSweep runs the add scenario for each hash strategy and returns one
// comparison per strategy. Hash sets are unchanged; only Bloom filter hashing varies.
func CompareHashSweep(cfg Config, strategies []bloom.Strategy) ([]Comparison, error) {
	if len(strategies) == 0 {
		return nil, fmt.Errorf("benchcompare: hash sweep requires at least one strategy")
	}
	out := make([]Comparison, 0, len(strategies))
	for _, strategy := range strategies {
		sweepCfg := cfg
		sweepCfg.Bloom = sweepCfg.Bloom.WithHashStrategy(strategy)
		cmp, err := compareAdd(sweepCfg)
		if err != nil {
			return nil, err
		}
		out = append(out, cmp)
	}
	return out, nil
}

// ParseHashStrategies parses comma-separated hash strategy names.
func ParseHashStrategies(raw string) ([]bloom.Strategy, error) {
	parts := strings.Split(raw, ",")
	strategies := make([]bloom.Strategy, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		s, err := bloom.ParseStrategy(part)
		if err != nil {
			return nil, fmt.Errorf("benchcompare: %w", err)
		}
		strategies = append(strategies, s)
	}
	if len(strategies) == 0 {
		return nil, fmt.Errorf("benchcompare: no hash strategies in %q", raw)
	}
	return strategies, nil
}
