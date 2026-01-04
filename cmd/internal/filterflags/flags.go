package filterflags

import (
	"flag"
	"fmt"

	"github.com/dannysecurity/bloomlab/bloom"
)

// Flags holds shared CLI options for Bloom filter demos.
type Flags struct {
	Capacity     *uint64
	FPR          *float64
	Hash         *string
	Seed         *uint64
	MinBits      *uint64
	MaxHashCount *uint
	CounterWidth *uint
}

// Register binds shared Bloom sizing and hash flags with the given default capacity.
func Register(defaultCapacity uint64) *Flags {
	return &Flags{
		Capacity:     flag.Uint64("n", defaultCapacity, "expected number of items"),
		FPR:          flag.Float64("p", 0.01, "target false positive rate"),
		Hash:         flag.String("hash", "fnv", "hash strategy: fnv, murmur3, xxhash"),
		Seed:         flag.Uint64("seed", 0, "hash seed for independent filters"),
		MinBits:      flag.Uint64("min-bits", 0, "minimum bit count for target sizing (0 = default)"),
		MaxHashCount: flag.Uint("max-k", 0, "maximum hash functions for target sizing (0 = default)"),
	}
}

// RegisterCounting binds the standard flags plus -counter-width for counting filters.
func RegisterCounting(defaultCapacity uint64) *Flags {
	f := Register(defaultCapacity)
	f.CounterWidth = flag.Uint("counter-width", 8, "per-bit counter width in bits: 8 or 16")
	return f
}

// Config builds a target-sized bloom.Config from parsed flag values.
func (f *Flags) Config() (bloom.Config, error) {
	opts, err := f.ConfigOptions()
	if err != nil {
		return bloom.Config{}, err
	}
	return bloom.TargetConfig(*f.Capacity, *f.FPR, opts...), nil
}

// ConfigOptions returns bloom.ConfigOption values for hash and sizing bounds.
func (f *Flags) ConfigOptions() ([]bloom.ConfigOption, error) {
	strategy, err := bloom.ParseStrategy(*f.Hash)
	if err != nil {
		return nil, err
	}
	opts := []bloom.ConfigOption{bloom.WithHash(strategy)}
	if *f.Seed != 0 {
		opts = append(opts, bloom.WithSeed(*f.Seed))
	}
	if *f.MinBits != 0 {
		opts = append(opts, bloom.WithMinBits(*f.MinBits))
	}
	if *f.MaxHashCount != 0 {
		opts = append(opts, bloom.WithMaxHashCount(uint(*f.MaxHashCount)))
	}
	if f.CounterWidth != nil {
		switch *f.CounterWidth {
		case 8:
		case 16:
			opts = append(opts, bloom.WithCounterWidth(16))
		default:
			return nil, fmt.Errorf("counter-width must be 8 or 16")
		}
	}
	return opts, nil
}
