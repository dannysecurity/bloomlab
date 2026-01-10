package bloom

import (
	"fmt"
	"strings"
	"testing"
)

func TestMeasureProbeOverlapBounded(t *testing.T) {
	const m = 4096
	const k = 8
	const samples = 5000

	keyFor := func(i int) []byte {
		return []byte(fmt.Sprintf("overlap-%d", i))
	}

	for _, strategy := range AllStrategies() {
		t.Run(strategy.String(), func(t *testing.T) {
			overlap := MeasureProbeOverlap(NewHasher(strategy, 0), m, k, samples, keyFor)
			if overlap.TotalProbes != samples*int(k) {
				t.Fatalf("total probes = %d, want %d", overlap.TotalProbes, samples*int(k))
			}
			if overlap.OverlapRate < 0 || overlap.OverlapRate > 1 {
				t.Fatalf("overlap rate = %f, want [0,1]", overlap.OverlapRate)
			}
			if overlap.DuplicateProbes > overlap.TotalProbes {
				t.Fatalf("duplicate probes %d exceed total %d", overlap.DuplicateProbes, overlap.TotalProbes)
			}
		})
	}
}

func TestMeasureDeriveNsPerOpPositive(t *testing.T) {
	keyFor := func(i int) []byte {
		return []byte(fmt.Sprintf("bench-%d", i))
	}
	ns := MeasureDeriveNsPerOp(NewHasher(HashFNV, 0), 256, keyFor)
	if ns <= 0 {
		t.Fatalf("expected positive ns/op, got %f", ns)
	}
}

func TestRecommendHasherPreferSpeed(t *testing.T) {
	opts := TuneOptions{
		M:       2048,
		K:       6,
		Samples: 2000,
		KeyFor: func(i int) []byte {
			return []byte(fmt.Sprintf("speed-%d", i))
		},
		PreferSpeed: true,
		ChiMargin:   100, // accept every strategy on chi²
	}
	report := RecommendHasher(opts, AllStrategies(), []uint64{0})
	if report.Best.NsPerDerive <= 0 {
		t.Fatalf("expected positive ns/op on best pick, got %f", report.Best.NsPerDerive)
	}

	fastest := report.Strategies[0]
	for _, score := range report.Strategies[1:] {
		if score.NsPerDerive < fastest.NsPerDerive {
			fastest = score
		}
	}
	if report.Best.Strategy != fastest.Strategy {
		t.Fatalf("prefer-speed picked %s (%.0f ns/op), want fastest %s (%.0f ns/op)",
			report.Best.Strategy, report.Best.NsPerDerive,
			fastest.Strategy, fastest.NsPerDerive)
	}
}

func TestRecommendHasherIncludesOverlapAndThroughput(t *testing.T) {
	opts := TuneOptions{
		M:       1024,
		K:       4,
		Samples: 1000,
		KeyFor: func(i int) []byte {
			return []byte(fmt.Sprintf("metrics-%d", i))
		},
	}
	report := RecommendHasher(opts, []Strategy{HashMurmur3, HashXXHash}, []uint64{0})
	for _, score := range report.Strategies {
		if score.NsPerDerive <= 0 {
			t.Fatalf("%s: expected positive ns/op", score.Strategy)
		}
		if score.Overlap.TotalProbes == 0 {
			t.Fatalf("%s: expected probe overlap measurement", score.Strategy)
		}
	}
	text := FormatTuningReport(report)
	if !strings.Contains(text, "OVERLAP%") || !strings.Contains(text, "NS/OP") {
		t.Fatal("report missing overlap or throughput columns")
	}
}
