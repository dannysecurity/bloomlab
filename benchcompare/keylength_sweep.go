package benchcompare

import (
	"fmt"
	"strconv"
	"strings"
)

// DefaultKeyLengthSweepLengths are key byte lengths compared when -sweep-keylen
// is set without custom -keylen-values.
var DefaultKeyLengthSweepLengths = []int{16, 64, 256, 1024}

// CompareKeyLengthSweep runs ScenarioAdd at each key length and returns one
// comparison per length. Bloom filter bytes/item stay near the theoretical
// minimum for fixed n and p; hash set footprint grows with stored key bytes.
func CompareKeyLengthSweep(cfg Config, lengths []int) ([]Comparison, error) {
	if len(lengths) == 0 {
		return nil, fmt.Errorf("benchcompare: key length sweep requires at least one length")
	}
	n := cfg.Bloom.ExpectedCapacity()
	out := make([]Comparison, 0, len(lengths))
	for _, keyLen := range lengths {
		if keyLen <= 0 {
			return nil, fmt.Errorf("benchcompare: key length must be > 0")
		}
		idxWidth := len(strconv.FormatUint(n-1, 10))
		minLen := idxWidth + 4 // "key-" prefix
		if keyLen < minLen {
			return nil, fmt.Errorf("benchcompare: key length %d too short for n=%d (need >= %d)", keyLen, n, minLen)
		}
		keys := makeKeysWithLength(n, keyLen)
		cmp, err := compareAddWithKeys(cfg, keys, keyLen)
		if err != nil {
			return nil, err
		}
		out = append(out, cmp)
	}
	return out, nil
}

// ParseKeyLengths parses comma-separated key byte lengths for a key-length sweep.
func ParseKeyLengths(raw string) ([]int, error) {
	parts := strings.Split(raw, ",")
	lengths := make([]int, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		n, err := strconv.Atoi(part)
		if err != nil {
			return nil, fmt.Errorf("benchcompare: invalid key length %q: %w", part, err)
		}
		lengths = append(lengths, n)
	}
	if len(lengths) == 0 {
		return nil, fmt.Errorf("benchcompare: no key lengths in %q", raw)
	}
	return lengths, nil
}
