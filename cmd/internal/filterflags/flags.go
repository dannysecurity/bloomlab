package filterflags

import (
	"flag"

	"github.com/dannysecurity/bloomlab/bloom"
)

// Flags holds shared CLI options for Bloom filter demos.
type Flags struct {
	Capacity *uint64
	FPR      *float64
	Hash     *string
	Seed     *uint64
}

// Register binds -n, -p, -hash, and -seed flags with the given default capacity.
func Register(defaultCapacity uint64) *Flags {
	return &Flags{
		Capacity: flag.Uint64("n", defaultCapacity, "expected number of items"),
		FPR:      flag.Float64("p", 0.01, "target false positive rate"),
		Hash:     flag.String("hash", "fnv", "hash strategy: fnv, murmur3"),
		Seed:     flag.Uint64("seed", 0, "hash seed for independent filters"),
	}
}

// Config builds a target-sized bloom.Config from parsed flag values.
func (f *Flags) Config() (bloom.Config, error) {
	strategy, err := bloom.ParseStrategy(*f.Hash)
	if err != nil {
		return bloom.Config{}, err
	}
	opts := []bloom.ConfigOption{bloom.WithHash(strategy)}
	if *f.Seed != 0 {
		opts = append(opts, bloom.WithSeed(*f.Seed))
	}
	return bloom.TargetConfig(*f.Capacity, *f.FPR, opts...), nil
}
