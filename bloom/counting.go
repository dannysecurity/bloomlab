package bloom

import (
	"errors"
)

var ErrCounterOverflow = errors.New("bloom: counter overflow")

// CountingFilter supports deletion by tracking per-bit counters instead of bits.
// Counters are uint8; overflow returns ErrCounterOverflow.
type CountingFilter struct {
	counters []uint8
	m        uint64
	k        uint
}

// NewCounting creates a counting Bloom filter with m bits and k hash functions.
func NewCounting(m uint64, k uint) (*CountingFilter, error) {
	if m == 0 {
		return nil, ErrInvalidCapacity
	}
	if k == 0 {
		k = 1
	}
	return &CountingFilter{
		counters: make([]uint8, m),
		m:        m,
		k:        k,
	}, nil
}

// NewCountingFromTarget sizes a counting filter like a standard Bloom filter.
func NewCountingFromTarget(expectedCapacity uint64, falsePositiveRate float64) (*CountingFilter, error) {
	if expectedCapacity == 0 {
		return nil, ErrInvalidCapacity
	}
	if falsePositiveRate <= 0 || falsePositiveRate >= 1 {
		return nil, ErrInvalidFPR
	}
	m := optimalM(expectedCapacity, falsePositiveRate)
	k := optimalK(m, expectedCapacity)
	return NewCounting(m, k)
}

// Add increments counters for key.
func (cf *CountingFilter) Add(key []byte) error {
	h1, h2 := deriveHashes(key, cf.m, cf.k)
	for i := uint(0); i < cf.k; i++ {
		idx := bitIndex(h1, h2, cf.m, i)
		if cf.counters[idx] == 255 {
			return ErrCounterOverflow
		}
		cf.counters[idx]++
	}
	return nil
}

// Remove decrements counters for key. Returns false if key was not present.
func (cf *CountingFilter) Remove(key []byte) bool {
	if !cf.Contains(key) {
		return false
	}
	h1, h2 := deriveHashes(key, cf.m, cf.k)
	for i := uint(0); i < cf.k; i++ {
		idx := bitIndex(h1, h2, cf.m, i)
		if cf.counters[idx] > 0 {
			cf.counters[idx]--
		}
	}
	return true
}

// Contains reports whether key might be in the set.
func (cf *CountingFilter) Contains(key []byte) bool {
	h1, h2 := deriveHashes(key, cf.m, cf.k)
	for i := uint(0); i < cf.k; i++ {
		idx := bitIndex(h1, h2, cf.m, i)
		if cf.counters[idx] == 0 {
			return false
		}
	}
	return true
}

// BitCount returns m.
func (cf *CountingFilter) BitCount() uint64 { return cf.m }

// HashCount returns k.
func (cf *CountingFilter) HashCount() uint { return cf.k }
