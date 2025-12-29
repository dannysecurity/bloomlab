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
	CounterWidth *uint
}

// Register binds -n, -p, -hash, and -seed flags with the given default capacity.
func Register(defaultCapacity uint64) *Flags {
	return &Flags{
		Capacity: flag.Uint64("n", defaultCapacity, "expected number of items"),
		FPR:      flag.Float64("p", 0.01, "target false positive rate"),
		Hash:     flag.String("hash", "fnv", "hash strategy: fnv, murmur3, xxhash"),
		Seed:     flag.Uint64("seed", 0, "hash seed for independent filters"),
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
	opts, err := f.configOptions()
	if err != nil {
		return bloom.Config{}, err
	}
	return bloom.TargetConfig(*f.Capacity, *f.FPR, opts...), nil
}

func (f *Flags) configOptions() ([]bloom.ConfigOption, error) {
	strategy, err := bloom.ParseStrategy(*f.Hash)
	if err != nil {
		return nil, err
	}
	opts := []bloom.ConfigOption{bloom.WithHash(strategy)}
	if *f.Seed != 0 {
		opts = append(opts, bloom.WithSeed(*f.Seed))
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
