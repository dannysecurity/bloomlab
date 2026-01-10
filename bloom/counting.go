package bloom

import "errors"

var ErrCounterOverflow = errors.New("bloom: counter overflow")

// CountingFilter supports deletion by tracking per-bit counters instead of bits.
// Counters default to uint8; use WithCounterWidth(16) or Config.WithCounterWidth(16) for a wider variant
// that tolerates more duplicate inserts before ErrCounterOverflow.
type CountingFilter struct {
	counters8  []uint8
	counters16 []uint16
	width      uint8
	m          uint64
	k          uint
	n          uint64 // approximate insert count (Add calls, not deduplicated)
	hasher     Hasher
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
	cf := &CountingFilter{
		m:      m,
		k:      k,
		width:  width,
		hasher: cfg.Hasher(),
	}
	if width == 16 {
		cf.counters16 = make([]uint16, m)
	} else {
		cf.counters8 = make([]uint8, m)
	}
	return cf, nil
}

// NewCounting creates a counting Bloom filter with m bits and k hash functions.
func NewCounting(m uint64, k uint) (*CountingFilter, error) {
	return NewCountingFilter(ExplicitConfig(m, k))
}

// NewCountingFromTarget sizes a counting filter like a standard Bloom filter.
func NewCountingFromTarget(expectedCapacity uint64, falsePositiveRate float64) (*CountingFilter, error) {
	return NewCountingFilter(TargetConfig(expectedCapacity, falsePositiveRate))
}

// CounterWidth returns the per-bit counter width in bits (8 or 16).
func (cf *CountingFilter) CounterWidth() uint8 { return cf.width }

// CounterLimit returns the maximum value a single counter can hold before Add
// returns ErrCounterOverflow (255 for 8-bit counters, 65535 for 16-bit).
func (cf *CountingFilter) CounterLimit() uint64 {
	if cf.width == 16 {
		return 65535
	}
	return 255
}

// Add increments counters for key.
func (cf *CountingFilter) Add(key []byte) error {
	h1, h2 := cf.hasher.Derive(key)
	for i := uint(0); i < cf.k; i++ {
		idx := bitIndex(h1, h2, cf.m, i)
		if err := cf.incCounter(idx); err != nil {
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
		cf.decCounter(idx)
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
		if cf.counterAt(idx) == 0 {
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
	if cf.width == 16 {
		return uint64(len(cf.counters16)) * 2
	}
	return uint64(len(cf.counters8))
}

// BitCount returns m.
func (cf *CountingFilter) BitCount() uint64 { return cf.m }

// HashCount returns k.
func (cf *CountingFilter) HashCount() uint { return cf.k }

// Clear resets all counters and insert tracking. m, k, and the hasher are unchanged.
func (cf *CountingFilter) Clear() {
	if cf.width == 16 {
		for i := range cf.counters16 {
			cf.counters16[i] = 0
		}
	} else {
		for i := range cf.counters8 {
			cf.counters8[i] = 0
		}
	}
	cf.n = 0
}

// MaxCounter returns the largest counter value in the filter. Values near the
// width limit indicate that further Adds of keys hashing to the same positions
// may return ErrCounterOverflow.
func (cf *CountingFilter) MaxCounter() uint64 {
	var max uint64
	if cf.width == 16 {
		for _, c := range cf.counters16 {
			if v := uint64(c); v > max {
				max = v
			}
		}
		return max
	}
	for _, c := range cf.counters8 {
		if v := uint64(c); v > max {
			max = v
		}
	}
	return max
}

// OccupiedCount returns the number of counters that are non-zero.
func (cf *CountingFilter) OccupiedCount() uint64 {
	var occupied uint64
	if cf.width == 16 {
		for _, c := range cf.counters16 {
			if c > 0 {
				occupied++
			}
		}
		return occupied
	}
	for _, c := range cf.counters8 {
		if c > 0 {
			occupied++
		}
	}
	return occupied
}

// FillRatio returns the fraction of counters that are non-zero.
func (cf *CountingFilter) FillRatio() float64 {
	return float64(cf.OccupiedCount()) / float64(cf.m)
}

func (cf *CountingFilter) counterAt(idx uint64) uint64 {
	if cf.width == 16 {
		return uint64(cf.counters16[idx])
	}
	return uint64(cf.counters8[idx])
}

func (cf *CountingFilter) incCounter(idx uint64) error {
	if cf.width == 16 {
		if cf.counters16[idx] == 65535 {
			return ErrCounterOverflow
		}
		cf.counters16[idx]++
		return nil
	}
	if cf.counters8[idx] == 255 {
		return ErrCounterOverflow
	}
	cf.counters8[idx]++
	return nil
}

func (cf *CountingFilter) decCounter(idx uint64) {
	if cf.width == 16 {
		if cf.counters16[idx] > 0 {
			cf.counters16[idx]--
		}
		return
	}
	if cf.counters8[idx] > 0 {
		cf.counters8[idx]--
	}
}
