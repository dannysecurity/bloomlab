package benchcompare

import (
	"fmt"
	"strconv"
	"strings"
)

// DefaultSizeSweepCounts are item counts compared when -sweep-size is set
// without custom -size-values.
var DefaultSizeSweepCounts = []uint64{10_000, 50_000, 100_000, 500_000}

// CompareSizeSweep runs ScenarioAdd at each item count and returns one comparison
// per size. Hash set footprint scales with n; Bloom filter bytes/item stays near
// the theoretical minimum for a fixed target FPR.
func CompareSizeSweep(cfg Config, counts []uint64) ([]Comparison, error) {
	if len(counts) == 0 {
		return nil, fmt.Errorf("benchcompare: size sweep requires at least one item count")
	}
	out := make([]Comparison, 0, len(counts))
	for _, n := range counts {
		if n == 0 {
			return nil, fmt.Errorf("benchcompare: size sweep count must be > 0")
		}
		sweepCfg := cfg
		sweepCfg.ItemCount = n
		cmp, err := compareAdd(sweepCfg)
		if err != nil {
			return nil, err
		}
		out = append(out, cmp)
	}
	return out, nil
}

// ParseSizeCounts parses comma-separated item counts for a size sweep.
func ParseSizeCounts(raw string) ([]uint64, error) {
	parts := strings.Split(raw, ",")
	counts := make([]uint64, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		n, err := strconv.ParseUint(part, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("benchcompare: invalid item count %q: %w", part, err)
		}
		counts = append(counts, n)
	}
	if len(counts) == 0 {
		return nil, fmt.Errorf("benchcompare: no item counts in %q", raw)
	}
	return counts, nil
}
