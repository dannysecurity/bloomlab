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

// Validate checks that target sizing inputs are usable.
func (s TargetSpec) Validate() error {
	if s.Capacity == 0 {
		return ErrInvalidCapacity
	}
	if s.FPR <= 0 || s.FPR >= 1 {
		return ErrInvalidFPR
	}
	return nil
}

// ExplicitSpec holds fixed m and k for explicit sizing.
type ExplicitSpec struct {
	Bits      uint64
	HashCount uint
}

// Validate checks that explicit sizing inputs are usable.
func (s ExplicitSpec) Validate() error {
	if s.Bits == 0 {
		return ErrInvalidBits
	}
	return nil
}

// ConfigFromTarget builds a Config from typed target sizing inputs.
// Use BuildConfig when validation before construction is required.
func ConfigFromTarget(spec TargetSpec, opts ...ConfigOption) Config {
	cfg := Config{
		ExpectedCapacity:  spec.Capacity,
		FalsePositiveRate: spec.FPR,
		MinBits:           spec.Bounds.MinBits,
		MaxHashCount:      spec.Bounds.MaxHashCount,
	}
	applyOptions(&cfg, opts)
	return cfg
}

// ConfigFromExplicit builds a Config from typed explicit sizing inputs.
// HashCount of zero is treated as one when sizing is resolved.
func ConfigFromExplicit(spec ExplicitSpec, opts ...ConfigOption) Config {
	cfg := Config{
		Bits:      spec.Bits,
		HashCount: spec.HashCount,
	}
	applyOptions(&cfg, opts)
	return cfg
}

// BuildConfig constructs and validates a Config from typed sizing specs.
// For SizingTarget, supply target and leave explicit zero-valued.
// For SizingExplicit, supply explicit and leave target zero-valued.
func BuildConfig(mode SizingMode, target TargetSpec, explicit ExplicitSpec, opts ...ConfigOption) (Config, error) {
	var cfg Config
	switch mode {
	case SizingTarget:
		if err := target.Validate(); err != nil {
			return Config{}, err
		}
		cfg = ConfigFromTarget(target, opts...)
	case SizingExplicit:
		if err := explicit.Validate(); err != nil {
			return Config{}, err
		}
		cfg = ConfigFromExplicit(explicit, opts...)
	default:
		return Config{}, fmt.Errorf("bloom: unknown sizing mode %v", mode)
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
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
