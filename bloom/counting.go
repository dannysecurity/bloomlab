package bloom

import "errors"

var ErrCounterOverflow = errors.New("bloom: counter overflow")

// CountingFilter supports deletion by tracking per-bit counters instead of bits.
// Counters default to uint8; use WithCounterWidth(16) or WithCounterWidth(32) for wider
// variants that tolerate more duplicate inserts before ErrCounterOverflow.
type CountingFilter struct {
	store  counterStore
	width  uint8
	m      uint64
	k      uint
	n      uint64 // approximate insert count (Add calls, not deduplicated)
	hasher Hasher
}

// NewCountingFilter constructs a CountingFilter from cfg.
func NewCountingFilter(cfg Config) (*CountingFilter, error) {
	if err := cfg.validateCounterWidth(); err != nil {
		return nil, err
	}
	m, k, err := cfg.Size()
	if err != nil {
		return nil, err
	}
	width := cfg.resolvedCounterWidth()
	store, err := newCounterStore(m, width)
	if err != nil {
		return nil, err
	}
	return &CountingFilter{
		store:  store,
		m:      m,
		k:      k,
		width:  width,
		hasher: cfg.Hasher(),
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

// CounterWidth returns the per-bit counter width in bits (8, 16, or 32).
func (cf *CountingFilter) CounterWidth() uint8 { return cf.width }

// CounterLimit returns the maximum value a single counter can hold before Add
// returns ErrCounterOverflow (255 for 8-bit, 65535 for 16-bit, 4294967295 for 32-bit).
func (cf *CountingFilter) CounterLimit() uint64 {
	return cf.store.limit()
}

// CounterHeadroom returns how many more increments the fullest counter can
// accept before Add returns ErrCounterOverflow.
func (cf *CountingFilter) CounterHeadroom() uint64 {
	max := cf.MaxCounter()
	limit := cf.CounterLimit()
	if max >= limit {
		return 0
	}
	return limit - max
}

// Add increments counters for key.
func (cf *CountingFilter) Add(key []byte) error {
	h1, h2 := cf.hasher.Derive(key)
	for i := uint(0); i < cf.k; i++ {
		idx := bitIndex(h1, h2, cf.m, i)
		if err := cf.store.inc(idx); err != nil {
			return err
		}
	}
	cf.n++
	return nil
}

// Remove decrements counters for key. Returns false if key was not present
// (all k probed counters are zero).
//
// Only call Remove for keys previously inserted with Add. If Contains returns
// true for a key that was never added (a false positive), Remove will still
// decrement counters and can introduce false negatives for other keys.
func (cf *CountingFilter) Remove(key []byte) bool {
	if !cf.Contains(key) {
		return false
	}
	h1, h2 := cf.hasher.Derive(key)
	for i := uint(0); i < cf.k; i++ {
		idx := bitIndex(h1, h2, cf.m, i)
		cf.store.dec(idx)
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
		if cf.store.at(idx) == 0 {
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

// CounterBytes returns the number of bytes backing the per-bit counters.
func (cf *CountingFilter) CounterBytes() uint64 {
	return cf.m * cf.store.bytesPerCounter()
}

// BitCount returns m.
func (cf *CountingFilter) BitCount() uint64 { return cf.m }

// HashCount returns k.
func (cf *CountingFilter) HashCount() uint { return cf.k }

// Clear resets all counters and insert tracking. m, k, and the hasher are unchanged.
func (cf *CountingFilter) Clear() {
	cf.store.clear()
	cf.n = 0
}

// MaxCounter returns the largest counter value in the filter. Values near the
// width limit indicate that further Adds of keys hashing to the same positions
// may return ErrCounterOverflow.
func (cf *CountingFilter) MaxCounter() uint64 {
	return cf.store.max()
}

// OccupiedCount returns the number of counters that are non-zero.
func (cf *CountingFilter) OccupiedCount() uint64 {
	return cf.store.occupied()
}

// FillRatio returns the fraction of counters that are non-zero.
func (cf *CountingFilter) FillRatio() float64 {
	return float64(cf.OccupiedCount()) / float64(cf.m)
}
