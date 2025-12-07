package bloom

import "errors"

var ErrCounterOverflow = errors.New("bloom: counter overflow")

// CountingFilter supports deletion by tracking per-bit counters instead of bits.
// Counters are uint8; overflow returns ErrCounterOverflow.
type CountingFilter struct {
	counters []uint8
	m        uint64
	k        uint
	n        uint64 // approximate insert count (Add calls, not deduplicated)
	hasher   Hasher
}

// NewCountingFilter constructs a CountingFilter from cfg.
func NewCountingFilter(cfg Config) (*CountingFilter, error) {
	m, k, err := cfg.Size()
	if err != nil {
		return nil, err
	}
	return &CountingFilter{
		counters: make([]uint8, m),
		m:        m,
		k:        k,
		hasher:   cfg.Hasher(),
	}, nil
}

// NewCounting creates a counting Bloom filter with m bits and k hash functions.
func NewCounting(m uint64, k uint) (*CountingFilter, error) {
	return NewCountingFilter(ExplicitConfig(m, k))
}

// NewCountingFromTarget sizes a counting filter like a standard Bloom filter.
func NewCountingFromTarget(expectedCapacity uint64, falsePositiveRate float64) (*CountingFilter, error) {
	return NewCountingFilter(TargetConfig(expectedCapacity, falsePositiveRate))
}

// Add increments counters for key.
func (cf *CountingFilter) Add(key []byte) error {
	h1, h2 := cf.hasher.Derive(key)
	for i := uint(0); i < cf.k; i++ {
		idx := bitIndex(h1, h2, cf.m, i)
		if cf.counters[idx] == 255 {
			return ErrCounterOverflow
		}
		cf.counters[idx]++
	}
	cf.n++
	return nil
}

// Remove decrements counters for key. Returns false if key was not present.
func (cf *CountingFilter) Remove(key []byte) bool {
	if !cf.Contains(key) {
		return false
	}
	h1, h2 := cf.hasher.Derive(key)
	for i := uint(0); i < cf.k; i++ {
		idx := bitIndex(h1, h2, cf.m, i)
		if cf.counters[idx] > 0 {
			cf.counters[idx]--
		}
	}
	if cf.n > 0 {
		cf.n--
	}
	return true
}

// Contains reports whether key might be in the set.
func (cf *CountingFilter) Contains(key []byte) bool {
	h1, h2 := cf.hasher.Derive(key)
	for i := uint(0); i < cf.k; i++ {
		idx := bitIndex(h1, h2, cf.m, i)
		if cf.counters[idx] == 0 {
			return false
		}
	}
	return true
}

// ApproximateCount returns the number of successful Add calls (not deduplicated).
func (cf *CountingFilter) ApproximateCount() uint64 { return cf.n }

// TheoryFPR returns the theoretical false positive rate at the current insert
// count, using the filter's m, k, and ApproximateCount().
func (cf *CountingFilter) TheoryFPR() float64 {
	return TheoryFalsePositiveRate(cf.n, cf.m, cf.k)
}

// BitCount returns m.
func (cf *CountingFilter) BitCount() uint64 { return cf.m }

// HashCount returns k.
func (cf *CountingFilter) HashCount() uint { return cf.k }

// Clear resets all counters and insert tracking. m, k, and the hasher are unchanged.
func (cf *CountingFilter) Clear() {
	for i := range cf.counters {
		cf.counters[i] = 0
	}
	cf.n = 0
}

// FillRatio returns the fraction of counters that are non-zero.
func (cf *CountingFilter) FillRatio() float64 {
	var occupied uint64
	for _, c := range cf.counters {
		if c > 0 {
			occupied++
		}
	}
	return float64(occupied) / float64(cf.m)
}
