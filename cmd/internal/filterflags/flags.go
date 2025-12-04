package filterflags

import (
	"flag"

	"github.com/dannysecurity/bloomlab/bloom"
)

// Flags holds shared CLI options for Bloom filter demos.
type Flags struct {
	Capacity *uint64
	FPR      *float64
}

// Register binds -n and -p flags with the given default capacity.
func Register(defaultCapacity uint64) *Flags {
	return &Flags{
		Capacity: flag.Uint64("n", defaultCapacity, "expected number of items"),
		FPR:      flag.Float64("p", 0.01, "target false positive rate"),
	}
}

// Config builds a target-sized bloom.Config from parsed flag values.
func (f *Flags) Config() bloom.Config {
	return bloom.TargetConfig(*f.Capacity, *f.FPR)
}
