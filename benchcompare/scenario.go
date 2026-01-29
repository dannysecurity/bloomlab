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
	// ScenarioContainsMixed measures lookup throughput with a configurable mix of
	// present and absent keys (see Config.LookupHitRatio).
	ScenarioContainsMixed Scenario = "contains_mixed"
	// ScenarioMixedStream simulates stream dedup: check each key, insert on first sight.
	ScenarioMixedStream Scenario = "mixed_stream"
	// ScenarioRemove measures delete throughput using a counting Bloom filter vs map delete.
	ScenarioRemove Scenario = "remove"
)

// AllScenarios lists workloads reported by Compare and the benchcompare CLI.
var AllScenarios = []Scenario{
	ScenarioAdd,
	ScenarioContainsHit,
	ScenarioContainsMiss,
	ScenarioContainsMixed,
	ScenarioMixedStream,
	ScenarioRemove,
}

// Config controls Bloom filter sizing and benchmark runtime options.
// Bloom holds the canonical bloom.FilterConfig; LookupRepeats and LookupHitRatio are
// benchcompare-only settings.
type Config struct {
	Bloom          bloom.FilterConfig
	LookupRepeats  int
	LookupHitRatio float64
}

// NewConfig wraps a validated bloom.FilterConfig with benchcompare defaults.
func NewConfig(bloomCfg bloom.FilterConfig) Config {
	return Config{
		Bloom:          bloomCfg,
		LookupRepeats:  1,
		LookupHitRatio: 0.5,
	}
}

// DefaultConfig returns settings aligned with bloom/bloom_bench_test.go.
func DefaultConfig() Config {
	return NewConfig(bloom.TargetFilter(100_000, 0.01))
}
