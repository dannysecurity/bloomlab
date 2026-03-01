package bloom

// splitMix64 advances a 64-bit state through the SplitMix64 sequence.
// It is used to derive uncorrelated seed neighbors during hash tuning sweeps.
func splitMix64(x uint64) uint64 {
	x += doubleHashSeedMix
	z := x
	z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9
	z = (z ^ (z >> 27)) * 0x94d049bb133111eb
	return z ^ (z >> 31)
}

// ExpandTuneSeeds appends splitMix64-derived neighbors for each base seed.
// steps is how many expansion values to add per base seed (the base seeds
// themselves are always retained). When steps <= 0, base is returned unchanged.
// The result is deduplicated while preserving first-seen order.
func ExpandTuneSeeds(base []uint64, steps int) []uint64 {
	if steps <= 0 || len(base) == 0 {
		return append([]uint64(nil), base...)
	}

	seen := make(map[uint64]struct{}, len(base)*(steps+1))
	out := make([]uint64, 0, len(base)*(steps+1))
	add := func(seed uint64) {
		if _, ok := seen[seed]; ok {
			return
		}
		seen[seed] = struct{}{}
		out = append(out, seed)
	}

	for _, seed := range base {
		add(seed)
		state := seed
		for i := 0; i < steps; i++ {
			state = splitMix64(state)
			add(state)
		}
	}
	return out
}

// ResolveTuneSeeds returns the seed ladder used for tuning. When seeds is empty,
// DefaultTuneSeeds is used. When expandSteps > 0, each base seed is expanded
// with splitMix64 neighbors via ExpandTuneSeeds.
func ResolveTuneSeeds(seeds []uint64, expandSteps int) []uint64 {
	if len(seeds) == 0 {
		seeds = DefaultTuneSeeds()
	}
	return ExpandTuneSeeds(seeds, expandSteps)
}
