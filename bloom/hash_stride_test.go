package bloom

import (
	"fmt"
	"testing"
)

func TestMeasureDoubleHashStride(t *testing.T) {
	const m = 997 // prime — gcd(h2, m) should usually be 1
	const k = 7
	const samples = 5000

	keyFor := func(i int) []byte {
		return []byte(fmt.Sprintf("stride-%d", i))
	}

	for _, strategy := range AllStrategies() {
		t.Run(strategy.String(), func(t *testing.T) {
			stride := MeasureDoubleHashStride(NewHasher(strategy, 0), m, k, samples, keyFor)
			if stride.Samples != samples {
				t.Fatalf("samples = %d, want %d", stride.Samples, samples)
			}
			if strategy == HashFNV {
				return
			}
			if !stride.StrideHealthy(0.05) {
				t.Fatalf("stride unhealthy: gcd-rate=%.2f%% short=%d mean-reach=%.1f",
					stride.GCDgtOneRate*100, stride.ShortCycleKeys, stride.MeanReachable)
			}
		})
	}
}

func TestMeasureH1H2Correlation(t *testing.T) {
	const samples = 5000
	keyFor := func(i int) []byte {
		return []byte(fmt.Sprintf("corr-%d-%d", i, i*7919))
	}

	for _, strategy := range AllStrategies() {
		t.Run(strategy.String(), func(t *testing.T) {
			corr := MeasureH1H2Correlation(NewHasher(strategy, 0), samples, keyFor)
			if corr.Samples != samples {
				t.Fatalf("samples = %d, want %d", corr.Samples, samples)
			}
			if strategy == HashFNV {
				return
			}
			if !corr.CorrelationBelow(0.25) {
				t.Fatalf("|r|=%.3f exceeds 0.25 for %s", corr.Pearson, strategy)
			}
		})
	}
}

func TestReachableProbeCount(t *testing.T) {
	got := reachableProbeCount(10, 3, 16, 8)
	if got != 8 {
		t.Fatalf("reachable = %d, want 8 distinct probes", got)
	}

	// h2 shares factor with m=12 -> at most 4 distinct positions
	got = reachableProbeCount(0, 4, 12, 12)
	if got > 4 {
		t.Fatalf("expected short cycle with gcd(h2,m)>1, got %d reachable", got)
	}
}

func TestGCDUint64(t *testing.T) {
	if gcdUint64(12, 8) != 4 {
		t.Fatal("expected gcd(12,8)=4")
	}
	if gcdUint64(17, 13) != 1 {
		t.Fatal("expected coprime gcd=1")
	}
}

func TestSinglePass128Derivation(t *testing.T) {
	key := []byte("single-pass-check")
	for _, strategy := range []Strategy{HashMurmur3, HashXXHash, HashWyhash} {
		t.Run(strategy.String(), func(t *testing.T) {
			h1, h2 := NewHasher(strategy, 42).Derive(key)
			if h2 == 0 {
				t.Fatal("h2 must be non-zero")
			}
			if h1 == h2 {
				t.Fatal("h1 and h2 should differ for double hashing")
			}
		})
	}
}
