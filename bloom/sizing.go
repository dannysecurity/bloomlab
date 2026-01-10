package bloom

import "fmt"

// SizingMode names how m (bits) and k (hash functions) are chosen.
type SizingMode int

const (
	// SizingTarget derives m and k from expected capacity and false positive rate.
	SizingTarget SizingMode = iota
	// SizingExplicit uses fixed Bits and HashCount from the configuration.
	SizingExplicit
)

// String returns a short label for CLI output and debugging.
func (m SizingMode) String() string {
	switch m {
	case SizingTarget:
		return "target"
	case SizingExplicit:
		return "explicit"
	default:
		return fmt.Sprintf("SizingMode(%d)", m)
	}
}

// SizingBounds caps target-based sizing. Zero fields use package defaults.
type SizingBounds struct {
	MinBits      uint64
	MaxHashCount uint
}

// DefaultSizingBounds returns the bounds applied when MinBits and MaxHashCount are unset.
func DefaultSizingBounds() SizingBounds {
	return SizingBounds{
		MinBits:      defaultMinBits,
		MaxHashCount: defaultMaxHashCount,
	}
}

// Resolved returns bounds with zero fields replaced by package defaults.
func (b SizingBounds) Resolved() SizingBounds {
	out := b
	if out.MinBits == 0 {
		out.MinBits = defaultMinBits
	}
	if out.MaxHashCount == 0 {
		out.MaxHashCount = defaultMaxHashCount
	}
	return out
}

// String summarizes the bounds for debugging and CLI output.
func (b SizingBounds) String() string {
	r := b.Resolved()
	return fmt.Sprintf("minBits=%d maxK=%d", r.MinBits, r.MaxHashCount)
}

// TargetSpec holds the inputs for target-based sizing.
type TargetSpec struct {
	Capacity uint64
	FPR      float64
	Bounds   SizingBounds
}

// ExplicitSpec holds fixed m and k for explicit sizing.
type ExplicitSpec struct {
	Bits      uint64
	HashCount uint
}

// Mode reports whether the configuration uses target or explicit sizing.
// Incomplete explicit configs (zero m with non-zero k) report SizingExplicit
// even though Validate rejects them until m is positive.
func (c Config) Mode() SizingMode {
	if c.isExplicitSizing() {
		return SizingExplicit
	}
	return SizingTarget
}

// Bounds returns the sizing bounds used in target mode.
func (c Config) Bounds() SizingBounds {
	return SizingBounds{
		MinBits:      c.MinBits,
		MaxHashCount: c.MaxHashCount,
	}
}

// Target returns target sizing inputs when Mode is SizingTarget.
func (c Config) Target() (TargetSpec, bool) {
	if c.Mode() != SizingTarget {
		return TargetSpec{}, false
	}
	return TargetSpec{
		Capacity: c.ExpectedCapacity,
		FPR:      c.FalsePositiveRate,
		Bounds:   c.Bounds(),
	}, true
}

// Explicit returns fixed sizing inputs when Mode is SizingExplicit.
func (c Config) Explicit() (ExplicitSpec, bool) {
	if c.Mode() != SizingExplicit {
		return ExplicitSpec{}, false
	}
	return ExplicitSpec{
		Bits:      c.Bits,
		HashCount: c.HashCount,
	}, true
}
