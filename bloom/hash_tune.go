package bloom

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"text/tabwriter"
)

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

// TuneOptionsFromConfig builds spread-measurement options from a resolved filter config.
func TuneOptionsFromConfig(cfg Config, samples int, keyPrefix string) (TuneOptions, error) {
	m, k, err := cfg.Size()
	if err != nil {
		return TuneOptions{}, err
	}
	if samples <= 0 {
		samples = 10_000
	}
	if keyPrefix == "" {
		keyPrefix = "tune"
	}
	prefix := keyPrefix
	return TuneOptions{
		M:       m,
		K:       k,
		Samples: samples,
		KeyFor: func(i int) []byte {
			return []byte(fmt.Sprintf("%s-%d", prefix, i))
		},
	}, nil
}

// RecommendHasherFromConfig is a convenience wrapper around RecommendHasher.
func RecommendHasherFromConfig(cfg Config, samples int, keyPrefix string, strategies []Strategy, seeds []uint64) (TuningReport, error) {
	opts, err := TuneOptionsFromConfig(cfg, samples, keyPrefix)
	if err != nil {
		return TuningReport{}, err
	}
	return RecommendHasher(opts, strategies, seeds), nil
}

// BestHashConfig returns the recommended hash settings from a tuning report.
func (r TuningReport) BestHashConfig() HashConfig {
	return HashConfig{Strategy: r.Best.Strategy, Seed: r.Best.Seed}
}

// ParseSeeds parses comma-separated decimal or 0x-prefixed hex seeds.
func ParseSeeds(raw string) ([]uint64, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	parts := strings.Split(raw, ",")
	seeds := make([]uint64, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		seed, err := parseSeedValue(part)
		if err != nil {
			return nil, fmt.Errorf("bloom: parse seed %q: %w", part, err)
		}
		seeds = append(seeds, seed)
	}
	if len(seeds) == 0 {
		return nil, fmt.Errorf("bloom: no seeds in %q", raw)
	}
	return seeds, nil
}

func parseSeedValue(raw string) (uint64, error) {
	base := 10
	if strings.HasPrefix(raw, "0x") || strings.HasPrefix(raw, "0X") {
		base = 16
		raw = raw[2:]
	}
	seed, err := strconv.ParseUint(raw, base, 64)
	if err != nil {
		return 0, err
	}
	return seed, nil
}

// FormatTuningReport renders a human-readable tuning summary.
func FormatTuningReport(report TuningReport) string {
	opts := report.Options
	var b strings.Builder
	fmt.Fprintf(&b, "Hash tuning for m=%d k=%d (%d probe keys)\n\n", opts.M, opts.K, opts.Samples)

	if len(report.Strategies) > 0 {
		fmt.Fprintln(&b, "Strategy comparison (lower chi² is more uniform):")
		tw := tabwriter.NewWriter(&b, 0, 0, 2, ' ', 0)
		fmt.Fprintln(tw, "STRATEGY\tSEED\tCHI²\tMIN\tMAX\tEMPTY")
		for _, score := range report.Strategies {
			spread := score.Spread
			fmt.Fprintf(tw, "%s\t%d\t%.1f\t%d\t%d\t%d\n",
				score.Strategy, score.Seed, spread.ChiSquared,
				spread.MinCount, spread.MaxCount, spread.EmptyBuckets)
		}
		tw.Flush()
	}

	if len(report.Candidates) > 0 && len(report.Strategies) > 0 {
		best := report.Candidates[0]
		fmt.Fprintf(&b, "\nBest seed for %s: %d (chi²=%.1f)\n",
			report.Strategies[0].Strategy, best.Seed, best.ChiSquared)
	}

	best := report.Best
	fmt.Fprintf(&b, "\nRecommended: -hash %s", best.Strategy)
	if best.Seed != 0 {
		fmt.Fprintf(&b, " -seed %d", best.Seed)
	}
	fmt.Fprintf(&b, " (chi²=%.1f, empty=%d)\n", best.Spread.ChiSquared, best.Spread.EmptyBuckets)
	return b.String()
}

// FormatTuningReportMarkdown renders the tuning summary as a markdown table.
func FormatTuningReportMarkdown(report TuningReport) string {
	opts := report.Options
	var b strings.Builder
	fmt.Fprintf(&b, "## Hash tuning\n\n")
	fmt.Fprintf(&b, "Layout `m=%d`, `k=%d`, `%d` probe keys.\n\n", opts.M, opts.K, opts.Samples)

	if len(report.Strategies) > 0 {
		fmt.Fprintln(&b, "| Strategy | Seed | Chi² | Min | Max | Empty |")
		fmt.Fprintln(&b, "| --- | ---: | ---: | ---: | ---: | ---: |")
		for _, score := range report.Strategies {
			spread := score.Spread
			fmt.Fprintf(&b, "| %s | %d | %.1f | %d | %d | %d |\n",
				score.Strategy, score.Seed, spread.ChiSquared,
				spread.MinCount, spread.MaxCount, spread.EmptyBuckets)
		}
	}

	best := report.Best
	fmt.Fprintf(&b, "\n**Recommended:** `-hash %s`", best.Strategy)
	if best.Seed != 0 {
		fmt.Fprintf(&b, " `-seed %d`", best.Seed)
	}
	fmt.Fprintf(&b, " (chi²=%.1f)\n", best.Spread.ChiSquared)
	return b.String()
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
