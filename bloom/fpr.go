package bloom

import (
	"fmt"
	"math"
)

// TheoryFillFraction returns the expected fraction of bits set to 1 after
// inserting n distinct keys, under the standard independence approximation.
//
// Each insert sets k bits; treating positions as independent, a bit remains
// clear with probability e^(-kn/m), so the fill fraction is:
//
//	f ≈ 1 - e^(-kn/m)
func TheoryFillFraction(n, m uint64, k uint) float64 {
	if m == 0 || k == 0 || n == 0 {
		return 0
	}
	return 1 - math.Exp(-float64(k)*float64(n)/float64(m))
}

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
	return math.Pow(TheoryFillFraction(n, m, k), float64(k))
}

// SizingPlan summarizes target-based Bloom filter sizing and the theoretical
// rates at the expected capacity.
type SizingPlan struct {
	ExpectedCapacity uint64
	TargetFPR        float64
	Bits             uint64  // m
	HashCount        uint    // k
	AchievedFPR      float64 // TheoryFalsePositiveRate at ExpectedCapacity
	FillFraction     float64 // TheoryFillFraction at ExpectedCapacity
}

// PlanSizing resolves m and k from expected capacity and target FPR, then
// computes the theoretical fill fraction and achieved FPR at capacity.
func PlanSizing(expectedCapacity uint64, targetFPR float64, opts ...ConfigOption) (SizingPlan, error) {
	cfg := TargetConfig(expectedCapacity, targetFPR, opts...)
	m, k, err := cfg.Size()
	if err != nil {
		return SizingPlan{}, err
	}
	n := expectedCapacity
	return SizingPlan{
		ExpectedCapacity: n,
		TargetFPR:        targetFPR,
		Bits:             m,
		HashCount:        k,
		AchievedFPR:      TheoryFalsePositiveRate(n, m, k),
		FillFraction:     TheoryFillFraction(n, m, k),
	}, nil
}

// String formats the sizing plan for CLI output and debugging.
func (p SizingPlan) String() string {
	bitsPerItem := float64(p.Bits) / float64(p.ExpectedCapacity)
	return fmt.Sprintf(
		"target n=%d p=%g -> m=%d (%.2f bits/item) k=%d\n"+
			"at capacity: fill≈%.3f (%.1f%%), theory FPR≈%.5f (%.3f%%)",
		p.ExpectedCapacity, p.TargetFPR, p.Bits, bitsPerItem, p.HashCount,
		p.FillFraction, p.FillFraction*100,
		p.AchievedFPR, p.AchievedFPR*100,
	)
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
