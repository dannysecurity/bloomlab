package bloom

import "math"

// TheoryFalsePositiveRate returns the theoretical false positive probability
// after inserting n distinct keys into a filter with m bits and k hash functions.
//
// Under the standard independence approximation, the fraction of bits still
// clear after n inserts is e^(-kn/m), so a query on an absent key hits all k
// probed bits with probability:
//
//	p ≈ (1 - e^(-kn/m))^k
func TheoryFalsePositiveRate(n, m uint64, k uint) float64 {
	if m == 0 || k == 0 || n == 0 {
		return 0
	}
	fractionSet := 1 - math.Exp(-float64(k)*float64(n)/float64(m))
	return math.Pow(fractionSet, float64(k))
}

// TheoryFPRAt returns the theoretical FPR after n inserts using the resolved
// m and k from the configuration.
func (c Config) TheoryFPRAt(n uint64) (float64, error) {
	m, k, err := c.Size()
	if err != nil {
		return 0, err
	}
	return TheoryFalsePositiveRate(n, m, k), nil
}
