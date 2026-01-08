package bloom

import "math"

// TuneOptions configures hash tuning analysis for a planned filter layout.
type TuneOptions struct {
	// M is the bit count (bucket count) used for spread measurement.
	M uint64
	// K is the number of hash probes per key.
	K uint
	// Samples is how many synthetic keys to probe.
	Samples int
	// KeyFor maps sample indices to representative keys for the workload.
	KeyFor func(i int) []byte
}

// SeedCandidate scores a single seed for bucket uniformity under double hashing.
type SeedCandidate struct {
	Seed       uint64
	Spread     BucketSpread
	ChiSquared float64
}

// StrategyScore ranks a hash strategy (optionally with a tuned seed).
type StrategyScore struct {
	Strategy Strategy
	Seed     uint64
	Spread   BucketSpread
}

// TuningReport summarizes hash tuning results for a filter layout.
type TuningReport struct {
	Options    TuneOptions
	Candidates []SeedCandidate
	BestSeed   uint64
	Strategies []StrategyScore
	Best       StrategyScore
}

// DefaultTuneSeeds returns a small ladder of seeds suitable for quick tuning sweeps.
func DefaultTuneSeeds() []uint64 {
	return []uint64{0, 1, 7, 42, 0xdeadbeef, 0xcafebabe, 0x9e3779b97f4a7c15}
}

// TuneSeed evaluates candidate seeds for a strategy and returns the lowest chi-squared pick.
func TuneSeed(strategy Strategy, opts TuneOptions, seeds []uint64) SeedCandidate {
	if len(seeds) == 0 {
		seeds = DefaultTuneSeeds()
	}
	best := SeedCandidate{Seed: seeds[0], ChiSquared: math.MaxFloat64}
	for _, seed := range seeds {
		spread := MeasureBucketSpread(NewHasher(strategy, seed), opts.M, opts.K, opts.Samples, opts.KeyFor)
		candidate := SeedCandidate{Seed: seed, Spread: spread, ChiSquared: spread.ChiSquared}
		if candidate.ChiSquared < best.ChiSquared {
			best = candidate
		}
	}
	return best
}

// CompareSeeds ranks every candidate seed for a strategy by chi-squared (lower is better).
func CompareSeeds(strategy Strategy, opts TuneOptions, seeds []uint64) []SeedCandidate {
	if len(seeds) == 0 {
		seeds = DefaultTuneSeeds()
	}
	out := make([]SeedCandidate, 0, len(seeds))
	for _, seed := range seeds {
		spread := MeasureBucketSpread(NewHasher(strategy, seed), opts.M, opts.K, opts.Samples, opts.KeyFor)
		out = append(out, SeedCandidate{Seed: seed, Spread: spread, ChiSquared: spread.ChiSquared})
	}
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j].ChiSquared < out[j-1].ChiSquared; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}

// RecommendHasher picks the best strategy/seed pair from the given strategies by chi-squared spread.
func RecommendHasher(opts TuneOptions, strategies []Strategy, seeds []uint64) TuningReport {
	if len(strategies) == 0 {
		strategies = AllStrategies()
	}
	if len(seeds) == 0 {
		seeds = DefaultTuneSeeds()
	}

	report := TuningReport{Options: opts}
	report.Candidates = CompareSeeds(strategies[0], opts, seeds)
	if len(report.Candidates) > 0 {
		report.BestSeed = report.Candidates[0].Seed
	}

	report.Strategies = make([]StrategyScore, 0, len(strategies))
	bestChi := math.MaxFloat64
	for _, strategy := range strategies {
		tuned := TuneSeed(strategy, opts, seeds)
		score := StrategyScore{Strategy: strategy, Seed: tuned.Seed, Spread: tuned.Spread}
		report.Strategies = append(report.Strategies, score)
		if tuned.ChiSquared < bestChi {
			bestChi = tuned.ChiSquared
			report.Best = score
		}
	}
	return report
}
