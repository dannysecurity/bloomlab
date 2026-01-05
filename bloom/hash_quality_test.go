package bloom

import (
	"fmt"
	"testing"
)

func TestMeasureBucketSpreadUniformity(t *testing.T) {
	const m = 4096
	const k = 8
	const samples = 20_000

	keyFor := func(i int) []byte {
		return []byte(fmt.Sprintf("quality-key-%d-%d", i, i*7919))
	}

	for _, strategy := range AllStrategies() {
		t.Run(strategy.String(), func(t *testing.T) {
			spread := MeasureBucketSpread(NewHasher(strategy, 0), m, k, samples, keyFor)
			if spread.Probes != samples*k {
				t.Fatalf("probes = %d, want %d", spread.Probes, samples*k)
			}
			if strategy == HashFNV {
				if spread.EmptyBuckets > int(m)/8 {
					t.Fatalf("%d empty buckets exceeds m/8", spread.EmptyBuckets)
				}
				return
			}
			if !spread.WithinSpreadTolerance(4) {
				t.Fatalf("spread [%d, %d] outside 4x of mean %.1f (chi²=%.1f)",
					spread.MinCount, spread.MaxCount, spread.MeanCount, spread.ChiSquared)
			}
		})
	}
}

func TestBestUniformStrategy(t *testing.T) {
	const m = 2048
	const k = 6
	const samples = 10_000

	keyFor := func(i int) []byte {
		return []byte(fmt.Sprintf("pick-%d", i))
	}

	best := BestUniformStrategy(m, k, samples, keyFor, AllStrategies())
	if best == 0 && len(AllStrategies()) > 0 {
		t.Fatalf("unexpected zero strategy")
	}

	spreads := CompareBucketSpread(m, k, samples, keyFor, AllStrategies()...)
	if len(spreads) != len(AllStrategies()) {
		t.Fatalf("got %d spreads, want %d", len(spreads), len(AllStrategies()))
	}
}

func TestBucketSpreadToleranceHelpers(t *testing.T) {
	mean := BucketSpread{MeanCount: 100, MinCount: 30, MaxCount: 350, ChiSquared: 500}
	if !mean.WithinSpreadTolerance(4) {
		t.Fatal("expected spread within 4x tolerance")
	}
	if mean.ChiSquaredBelow(600) != true {
		t.Fatal("expected chi-squared below 600")
	}
	if mean.ChiSquaredBelow(400) {
		t.Fatal("expected chi-squared above 400")
	}

	degenerate := BucketSpread{MeanCount: 0}
	if degenerate.WithinSpreadTolerance(4) {
		t.Fatal("zero mean should fail tolerance check")
	}
}
