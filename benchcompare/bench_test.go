package benchcompare

import (
	"fmt"
	"testing"

	"github.com/dannysecurity/bloomlab/bloom"
)

func BenchmarkCompareAdd(b *testing.B) {
	cfg := smallBenchConfig()
	for i := 0; i < b.N; i++ {
		if _, err := compareAdd(cfg); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCompareContainsHit(b *testing.B) {
	cfg := smallBenchConfig()
	for i := 0; i < b.N; i++ {
		if _, err := compareContainsHit(cfg); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCompareContainsMiss(b *testing.B) {
	cfg := smallBenchConfig()
	for i := 0; i < b.N; i++ {
		if _, err := compareContainsMiss(cfg); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCompareContainsMixed(b *testing.B) {
	cfg := smallBenchConfig()
	cfg.LookupHitRatio = 0.5
	for i := 0; i < b.N; i++ {
		if _, err := compareContainsMixed(cfg); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCompareMixedStream(b *testing.B) {
	cfg := smallBenchConfig()
	for i := 0; i < b.N; i++ {
		if _, err := compareMixedStream(cfg); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCompareRemove(b *testing.B) {
	cfg := smallBenchConfig()
	for i := 0; i < b.N; i++ {
		if _, err := compareRemove(cfg); err != nil {
			b.Fatal(err)
		}
	}
}

// End-to-end benchmarks mirror cmd/benchcompare workloads at default sizing.
func BenchmarkCompareAllDefault(b *testing.B) {
	cfg := DefaultConfig()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := Compare(cfg); err != nil {
			b.Fatal(err)
		}
	}
}

func smallBenchConfig() Config {
	return Config{Bloom: bloom.TargetFilter(5_000, 0.01),
		LookupRepeats:     2,
	}
}

func BenchmarkCompareSizeSweep(b *testing.B) {
	cfg := smallBenchConfig()
	counts := []uint64{1_000, 2_000, 5_000}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := CompareSizeSweep(cfg, counts); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCompareLookupMixSweep(b *testing.B) {
	cfg := smallBenchConfig()
	ratios := []float64{0, 0.5, 1.0}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := CompareLookupMixSweep(cfg, ratios); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkCompareKeyLengthSweep(b *testing.B) {
	cfg := smallBenchConfig()
	lengths := []int{16, 64, 256}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := CompareKeyLengthSweep(cfg, lengths); err != nil {
			b.Fatal(err)
		}
	}
}

// ReportMetrics emits custom bench metrics so `go test -bench=ReportMetrics`
// can compare bloom vs hash set side by side without parsing table output.
func BenchmarkReportMetrics(b *testing.B) {
	cfg := DefaultConfig()
	results, err := Compare(cfg)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, cmp := range results {
			_ = cmp.SummaryLine()
		}
	}
	b.StopTimer()
	for _, cmp := range results {
		label := fmt.Sprintf("%s-bloom-ns", cmp.Scenario)
		b.ReportMetric(cmp.Bloom.NsPerOp, label)
		label = fmt.Sprintf("%s-hashset-ns", cmp.Scenario)
		b.ReportMetric(cmp.HashSet.NsPerOp, label)
		label = fmt.Sprintf("%s-space-ratio", cmp.Scenario)
		b.ReportMetric(cmp.SpaceRatio(), label)
		label = fmt.Sprintf("%s-alloc-ratio", cmp.Scenario)
		b.ReportMetric(cmp.AllocRatio(), label)
	}
}
