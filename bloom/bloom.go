package bloom

// Filter is a classic Bloom filter backed by a bit slice.
type Filter struct {
	bits []byte
	m    uint64 // number of bits
	k    uint   // number of hash functions
	n    uint64 // approximate insert count
}

// NewFilter constructs a Filter from cfg.
func NewFilter(cfg Config) (*Filter, error) {
	m, k, err := cfg.Size()
	if err != nil {
		return nil, err
	}
	return &Filter{
		bits: make([]byte, (m+7)/8),
		m:    m,
		k:    k,
	}, nil
}

// New creates a Bloom filter sized for expectedCapacity items at the
// given falsePositiveRate. m and k are computed using the standard formulas.
func New(expectedCapacity uint64, falsePositiveRate float64) (*Filter, error) {
	return NewFilter(TargetConfig(expectedCapacity, falsePositiveRate))
}

// Add inserts key into the filter.
func (f *Filter) Add(key []byte) {
	h1, h2 := deriveHashes(key, f.m, f.k)
	for i := uint(0); i < f.k; i++ {
		idx := bitIndex(h1, h2, f.m, i)
		f.bits[idx/8] |= 1 << (idx % 8)
	}
	f.n++
}

// Contains reports whether key might be in the set (false positives possible).
func (f *Filter) Contains(key []byte) bool {
	h1, h2 := deriveHashes(key, f.m, f.k)
	for i := uint(0); i < f.k; i++ {
		idx := bitIndex(h1, h2, f.m, i)
		if f.bits[idx/8]&(1<<(idx%8)) == 0 {
			return false
		}
	}
	return true
}

// ApproximateCount returns the number of Add calls (not deduplicated).
func (f *Filter) ApproximateCount() uint64 { return f.n }

// BitCount returns m (total bits allocated).
func (f *Filter) BitCount() uint64 { return f.m }

// HashCount returns k (number of hash functions).
func (f *Filter) HashCount() uint { return f.k }

// FillRatio returns the fraction of bits set to 1.
func (f *Filter) FillRatio() float64 {
	var set uint64
	for _, b := range f.bits {
		set += uint64(popcount(b))
	}
	return float64(set) / float64(f.m)
}

func popcount(b byte) int {
	n := 0
	for b != 0 {
		n += int(b & 1)
		b >>= 1
	}
	return n
}
