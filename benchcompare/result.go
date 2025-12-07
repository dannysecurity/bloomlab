package benchcompare

import "fmt"

// BackendResult captures throughput and space for one data structure.
type BackendResult struct {
	NsPerOp         float64
	BytesPerItem    float64
	AllocsPerOp     float64
	TheoryFPR       float64 // Bloom only; zero for hash sets
	FalsePositives  int     // empirical misses treated as hits in mixed stream
}

// Comparison holds paired measurements for a single scenario.
type Comparison struct {
	Scenario Scenario
	Bloom    BackendResult
	HashSet  BackendResult
}

// SpeedRatio returns hash-set ns/op divided by bloom ns/op. Values above 1 mean
// the Bloom filter was faster for that workload.
func (c Comparison) SpeedRatio() float64 {
	if c.Bloom.NsPerOp <= 0 {
		return 0
	}
	return c.HashSet.NsPerOp / c.Bloom.NsPerOp
}

// SpaceRatio returns hash-set bytes/item divided by bloom bytes/item. Values above
// 1 mean the Bloom filter used less memory per item.
func (c Comparison) SpaceRatio() float64 {
	if c.Bloom.BytesPerItem <= 0 {
		return 0
	}
	return c.HashSet.BytesPerItem / c.Bloom.BytesPerItem
}

// SummaryLine returns a one-line human-readable comparison.
func (c Comparison) SummaryLine() string {
	return fmt.Sprintf("%s: bloom %.0f ns/op (%.1f B/item) vs hashset %.0f ns/op (%.1f B/item) — %.2fx faster, %.1fx smaller",
		c.Scenario,
		c.Bloom.NsPerOp, c.Bloom.BytesPerItem,
		c.HashSet.NsPerOp, c.HashSet.BytesPerItem,
		c.SpeedRatio(), c.SpaceRatio(),
	)
}
