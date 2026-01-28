package filterflags

import (
	"flag"
	"fmt"

	"github.com/dannysecurity/bloomlab/bloom"
)

// DefaultCapacity is the shared default for -n when a CLI does not override it.
const DefaultCapacity uint64 = 10_000

// Flags holds shared CLI options for Bloom filter demos.
type Flags struct {
	Capacity     *uint64
	FPR          *float64
	Bits         *uint64
	HashCount    *uint
	Hash         *string
	Seed         *uint64
	MinBits      *uint64
	MaxHashCount *uint
	CounterWidth *uint
}

// Register binds shared Bloom sizing and hash flags with the given default capacity.
func Register(defaultCapacity uint64) *Flags {
	return &Flags{
		Capacity:     flag.Uint64("n", defaultCapacity, "expected number of items (target sizing)"),
		FPR:          flag.Float64("p", 0.01, "target false positive rate (target sizing)"),
		Bits:         flag.Uint64("m", 0, "fixed bit count for explicit sizing (0 = derive from -n and -p)"),
		HashCount:    flag.Uint("k", 0, "fixed hash function count for explicit sizing (0 = default at construction)"),
		Hash:         flag.String("hash", "fnv", "hash strategy: fnv, murmur3, xxhash, wyhash, highway"),
		Seed:         flag.Uint64("seed", 0, "hash seed for independent filters"),
		MinBits:      flag.Uint64("min-bits", 0, "minimum bit count for target sizing (0 = default)"),
		MaxHashCount: flag.Uint("max-k", 0, "maximum hash functions for target sizing (0 = default)"),
	}
}

// RegisterCounting binds the standard flags plus -counter-width for counting filters.
func RegisterCounting(defaultCapacity uint64) *Flags {
	f := Register(defaultCapacity)
	f.CounterWidth = flag.Uint("counter-width", 8, "per-bit counter width in bits: 8, 16, or 32")
	return f
}

// FilterConfig builds a bloom.FilterConfig from parsed flag values. When -m is set,
// explicit sizing is used; otherwise target sizing derives m and k from -n and -p.
func (f *Flags) FilterConfig() (bloom.FilterConfig, error) {
	hash, err := f.hashConfig()
	if err != nil {
		return bloom.FilterConfig{}, err
	}

	if *f.Bits != 0 {
		fc, err := bloom.BuildFilterConfig(bloom.SizingExplicit, bloom.TargetSpec{}, bloom.ExplicitSpec{
			Bits:      *f.Bits,
			HashCount: uint(*f.HashCount),
		})
		if err != nil {
			return bloom.FilterConfig{}, err
		}
		fc.Hash = hash
		return fc, nil
	}
	if *f.HashCount != 0 {
		return bloom.FilterConfig{}, fmt.Errorf("explicit sizing requires -m (bit count); -k without -m is invalid")
	}

	fc, err := bloom.BuildFilterConfig(bloom.SizingTarget, bloom.TargetSpec{
		Capacity: *f.Capacity,
		FPR:      *f.FPR,
		Bounds: bloom.SizingBounds{
			MinBits:      *f.MinBits,
			MaxHashCount: uint(*f.MaxHashCount),
		},
	}, bloom.ExplicitSpec{})
	if err != nil {
		return bloom.FilterConfig{}, err
	}
	fc.Hash = hash
	return fc, nil
}

// CountingConfig builds a bloom.CountingConfig from parsed flag values.
func (f *Flags) CountingConfig() (bloom.CountingConfig, error) {
	fc, err := f.FilterConfig()
	if err != nil {
		return bloom.CountingConfig{}, err
	}
	cc := bloom.CountingConfig{Filter: fc}
	if f.CounterWidth == nil {
		return cc, nil
	}
	switch *f.CounterWidth {
	case 8:
	case 16, 32:
		cc.CounterWidth = uint8(*f.CounterWidth)
	default:
		return bloom.CountingConfig{}, fmt.Errorf("counter-width must be 8, 16, or 32")
	}
	if err := cc.Validate(); err != nil {
		return bloom.CountingConfig{}, err
	}
	return cc, nil
}

// Config builds a bloom.Config from parsed flag values. Prefer FilterConfig when
// constructing filters directly; Config remains for legacy call sites.
func (f *Flags) Config() (bloom.Config, error) {
	fc, err := f.FilterConfig()
	if err != nil {
		return bloom.Config{}, err
	}
	cfg := fc.Config()
	if f.CounterWidth != nil && *f.CounterWidth != 8 {
		switch *f.CounterWidth {
		case 16, 32:
			cfg.CounterWidth = uint8(*f.CounterWidth)
		default:
			return bloom.Config{}, fmt.Errorf("counter-width must be 8, 16, or 32")
		}
	}
	return cfg, nil
}

// ConfigOptions returns bloom.ConfigOption values for hash and sizing bounds.
// Deprecated: new code should use FilterConfig, which applies bounds once via TargetSpec.
func (f *Flags) ConfigOptions() ([]bloom.ConfigOption, error) {
	strategy, err := bloom.ParseStrategy(*f.Hash)
	if err != nil {
		return nil, err
	}
	opts := []bloom.ConfigOption{bloom.WithHash(strategy)}
	if *f.Seed != 0 {
		opts = append(opts, bloom.WithSeed(*f.Seed))
	}
	if *f.MinBits != 0 || *f.MaxHashCount != 0 {
		opts = append(opts, bloom.WithSizingBounds(bloom.SizingBounds{
			MinBits:      *f.MinBits,
			MaxHashCount: uint(*f.MaxHashCount),
		}))
	}
	if f.CounterWidth != nil {
		switch *f.CounterWidth {
		case 8:
		case 16, 32:
			opts = append(opts, bloom.WithCounterWidth(uint8(*f.CounterWidth)))
		default:
			return nil, fmt.Errorf("counter-width must be 8, 16, or 32")
		}
	}
	return opts, nil
}

func (f *Flags) hashConfig() (bloom.HashConfig, error) {
	strategy, err := bloom.ParseStrategy(*f.Hash)
	if err != nil {
		return bloom.HashConfig{}, err
	}
	return bloom.HashConfig{Strategy: strategy, Seed: *f.Seed}, nil
}
