package bloom

import (
	"fmt"
	"testing"
)

func TestTuneSeedPicksLowerChiSquared(t *testing.T) {
	opts := TuneOptions{
		M:       2048,
		K:       6,
		Samples: 5000,
		KeyFor: func(i int) []byte {
			return []byte(fmt.Sprintf("tune-%d", i))
		},
	}
	seeds := []uint64{0, 42, 99}

	best := TuneSeed(HashMurmur3, opts, seeds)
	ranked := CompareSeeds(HashMurmur3, opts, seeds)
	if len(ranked) != len(seeds) {
		t.Fatalf("got %d ranked seeds, want %d", len(ranked), len(seeds))
	}
	if best.Seed != ranked[0].Seed {
		t.Fatalf("TuneSeed picked seed %d, CompareSeeds best is %d", best.Seed, ranked[0].Seed)
	}
	if best.ChiSquared != ranked[0].ChiSquared {
		t.Fatalf("chi-squared mismatch: %f vs %f", best.ChiSquared, ranked[0].ChiSquared)
	}
}

func TestRecommendHasherIncludesAllStrategies(t *testing.T) {
	opts := TuneOptions{
		M:       1024,
		K:       4,
		Samples: 3000,
		KeyFor: func(i int) []byte {
			return []byte(fmt.Sprintf("rec-%d", i*17))
		},
	}
	report := RecommendHasher(opts, AllStrategies(), []uint64{0, 7})
	if len(report.Strategies) != len(AllStrategies()) {
		t.Fatalf("got %d strategy scores, want %d", len(report.Strategies), len(AllStrategies()))
	}
	if report.Best.Strategy == 0 && report.Best.Spread.ChiSquared == 0 {
		t.Fatal("expected non-zero best strategy score")
	}
	if report.Best.Strategy.String() == "" {
		t.Fatal("best strategy should have a name")
	}
	if len(report.Candidates) != 2 {
		t.Fatalf("got %d seed candidates, want 2", len(report.Candidates))
	}
}

func TestHighwayHashDistribution(t *testing.T) {
	const m = 4096
	const k = 8
	const samples = 20_000

	keyFor := func(i int) []byte {
		return []byte(fmt.Sprintf("highway-key-%d", i))
	}

	spread := MeasureBucketSpread(NewHasher(HashHighway, 0), m, k, samples, keyFor)
	if !spread.WithinSpreadTolerance(4) {
		t.Fatalf("highway spread [%d, %d] outside 4x of mean %.1f (chi²=%.1f)",
			spread.MinCount, spread.MaxCount, spread.MeanCount, spread.ChiSquared)
	}
}

func TestHighwaySeedAffectsOutput(t *testing.T) {
	key := []byte("highway-seed-check")
	a1, a2 := NewHasher(HashHighway, 0).Derive(key)
	b1, b2 := NewHasher(HashHighway, 99).Derive(key)
	if a1 == b1 && a2 == b2 {
		t.Fatal("expected different hashes for different seeds")
	}
}

func TestDefaultTuneSeedsNonEmpty(t *testing.T) {
	seeds := DefaultTuneSeeds()
	if len(seeds) < 3 {
		t.Fatalf("expected several default seeds, got %d", len(seeds))
	}
}
