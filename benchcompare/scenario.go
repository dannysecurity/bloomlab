package benchcompare

import "github.com/dannysecurity/bloomlab/bloom"

// Scenario names a membership workload to compare between a Bloom filter and
// a hash set.
type Scenario string

const (
	// ScenarioAdd measures insert throughput on distinct keys.
	ScenarioAdd Scenario = "add"
	// ScenarioContainsHit measures lookup throughput for keys known to be present.
	ScenarioContainsHit Scenario = "contains_hit"
	// ScenarioContainsMiss measures lookup throughput for keys known to be absent.
	ScenarioContainsMiss Scenario = "contains_miss"
	// ScenarioMixedStream simulates stream dedup: check each key, insert on first sight.
	ScenarioMixedStream Scenario = "mixed_stream"
)

// AllScenarios lists workloads reported by Compare and the benchcompare CLI.
var AllScenarios = []Scenario{
	ScenarioAdd,
	ScenarioContainsHit,
	ScenarioContainsMiss,
	ScenarioMixedStream,
}

// Config controls sizing and iteration count for a comparison run.
type Config struct {
	// ItemCount is the number of distinct keys inserted (or probed for lookups).
	ItemCount uint64
	// FalsePositiveRate sizes the Bloom filter via bloom.TargetConfig.
	FalsePositiveRate float64
	// LookupRepeats is how many times each lookup key is queried per scenario.
	LookupRepeats int
	// Hash selects the Bloom filter hash family and seed (ignored by hash sets).
	Hash bloom.HashConfig
}

// bloomOptions returns functional options for bloom.TargetConfig.
func (cfg Config) bloomOptions() []bloom.ConfigOption {
	var opts []bloom.ConfigOption
	if cfg.Hash.Strategy != 0 {
		opts = append(opts, bloom.WithHash(cfg.Hash.Strategy))
	}
	if cfg.Hash.Seed != 0 {
		opts = append(opts, bloom.WithSeed(cfg.Hash.Seed))
	}
	return opts
}

func (cfg Config) targetBloomConfig() bloom.Config {
	return bloom.TargetConfig(cfg.ItemCount, cfg.FalsePositiveRate, cfg.bloomOptions()...)
}

// DefaultConfig returns settings aligned with bloom/bloom_bench_test.go.
func DefaultConfig() Config {
	return Config{
		ItemCount:         100_000,
		FalsePositiveRate: 0.01,
		LookupRepeats:     4,
	}
}
