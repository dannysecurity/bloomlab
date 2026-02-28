package bloom

import (
	"fmt"
	"strings"
	"testing"
)

func TestTuneSeedPicksLowestScore(t *testing.T) {
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
	if best.Score != ranked[0].Score {
		t.Fatalf("score mismatch: %f vs %f", best.Score, ranked[0].Score)
	}
	for i := 1; i < len(ranked); i++ {
		if ranked[i].Score < ranked[i-1].Score {
			t.Fatalf("CompareSeeds not sorted by score at index %d", i)
		}
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

func TestParseSeeds(t *testing.T) {
	seeds, err := ParseSeeds("0, 42, 0xdeadbeef")
	if err != nil {
		t.Fatal(err)
	}
	if len(seeds) != 3 {
		t.Fatalf("got %d seeds, want 3", len(seeds))
	}
	if seeds[2] != 0xdeadbeef {
		t.Fatalf("hex seed = %#x, want 0xdeadbeef", seeds[2])
	}

	if _, err := ParseSeeds("not-a-number"); err == nil {
		t.Fatal("expected error for invalid seed")
	}
	if _, err := ParseSeeds(""); err != nil {
		t.Fatal("empty seeds should return nil slice without error")
	}
}

func TestTuneOptionsFromConfig(t *testing.T) {
	cfg := TargetConfig(5000, 0.01)
	opts, err := TuneOptionsFromConfig(cfg, 1000, "probe")
	if err != nil {
		t.Fatal(err)
	}
	m, k, err := cfg.Size()
	if err != nil {
		t.Fatal(err)
	}
	if opts.M != m || opts.K != k || opts.Samples != 1000 {
		t.Fatalf("opts = m=%d k=%d samples=%d, want m=%d k=%d samples=1000", opts.M, opts.K, opts.Samples, m, k)
	}
	key := opts.KeyFor(7)
	if string(key) != "probe-7" {
		t.Fatalf("KeyFor(7) = %q, want probe-7", key)
	}
}

func TestRecommendHasherFromConfig(t *testing.T) {
	cfg := TargetConfig(2000, 0.01)
	report, err := RecommendHasherFromConfig(cfg, 2000, "cfg", AllStrategies(), []uint64{0, 7})
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Strategies) != len(AllStrategies()) {
		t.Fatalf("got %d strategies, want %d", len(report.Strategies), len(AllStrategies()))
	}
	hashCfg := report.BestHashConfig()
	if hashCfg.Strategy != report.Best.Strategy || hashCfg.Seed != report.Best.Seed {
		t.Fatalf("BestHashConfig = %+v, want strategy=%v seed=%d", hashCfg, report.Best.Strategy, report.Best.Seed)
	}
}

func TestFormatTuningReport(t *testing.T) {
	opts := TuneOptions{
		M:       1024,
		K:       4,
		Samples: 1000,
		KeyFor: func(i int) []byte {
			return []byte(fmt.Sprintf("fmt-%d", i))
		},
	}
	report := RecommendHasher(opts, []Strategy{HashMurmur3, HashXXHash}, []uint64{0, 42})
	text := FormatTuningReport(report)
	if !strings.Contains(text, "Recommended: -hash") {
		t.Fatal("report missing recommendation line")
	}
	if !strings.Contains(text, "Seed ranking") {
		t.Fatal("report missing seed ranking table")
	}
	if !strings.Contains(text, "murmur3") || !strings.Contains(text, "xxhash") {
		t.Fatal("report missing strategy rows")
	}
	md := FormatTuningReportMarkdown(report)
	if !strings.Contains(md, "| Strategy |") {
		t.Fatal("markdown report missing table header")
	}
}
