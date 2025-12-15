package bloom

import (
	"math"
	"testing"
	"testing/quick"
)

// empiricalFilter abstracts membership operations for false-positive probes.
type empiricalFilter struct {
	add      func(key []byte) error
	contains func(key []byte) bool
}

func encodeProbeKey(i int) []byte {
	return []byte{byte(i >> 8), byte(i), byte(i >> 16)}
}

// measureEmpiricalFPR inserts n distinct keys, then probes trials absent keys.
func measureEmpiricalFPR(f empiricalFilter, n, trials int) (rate float64, err error) {
	for i := 0; i < n; i++ {
		if err := f.add(encodeProbeKey(i)); err != nil {
			return 0, err
		}
	}
	falsePositives := 0
	for i := n; i < n+trials; i++ {
		if f.contains(encodeProbeKey(i)) {
			falsePositives++
		}
	}
	return float64(falsePositives) / float64(trials), nil
}

func assertEmpiricalFPR(t *testing.T, f empiricalFilter, n, trials int, maxRate float64) {
	t.Helper()
	rate, err := measureEmpiricalFPR(f, n, trials)
	if err != nil {
		t.Fatal(err)
	}
	if rate > maxRate {
		t.Errorf("empirical false positive rate %.4f exceeds max %.4f", rate, maxRate)
	}
}

func newStandardEmpiricalFilter(cfg Config) (empiricalFilter, error) {
	filter, err := NewFilter(cfg)
	if err != nil {
		return empiricalFilter{}, err
	}
	return empiricalFilter{
		add: func(key []byte) error {
			filter.Add(key)
			return nil
		},
		contains: filter.Contains,
	}, nil
}

func newCountingEmpiricalFilter(cfg Config) (empiricalFilter, error) {
	cf, err := NewCountingFilter(cfg)
	if err != nil {
		return empiricalFilter{}, err
	}
	return empiricalFilter{
		add:      cf.Add,
		contains: cf.Contains,
	}, nil
}

func TestEmpiricalFPR(t *testing.T) {
	const n = 5000
	const trials = 5000

	tests := []struct {
		name    string
		build   func(Config) (empiricalFilter, error)
		cfg     Config
		maxRate float64
	}{
		{
			name:    "standard filter",
			build:   newStandardEmpiricalFilter,
			cfg:     TargetConfig(n, 0.01),
			maxRate: 0.05,
		},
		{
			name:    "counting filter",
			build:   newCountingEmpiricalFilter,
			cfg:     TargetConfig(n, 0.01),
			maxRate: 0.05,
		},
		{
			name:    "fnv strategy",
			build:   newStandardEmpiricalFilter,
			cfg:     TargetConfig(n, 0.01, WithHash(HashFNV)),
			maxRate: 0.05,
		},
		{
			name:    "murmur3 strategy",
			build:   newStandardEmpiricalFilter,
			cfg:     TargetConfig(n, 0.01, WithHash(HashMurmur3)),
			maxRate: 0.05,
		},
		{
			name:    "tight fpr",
			build:   newStandardEmpiricalFilter,
			cfg:     TargetConfig(2000, 0.001),
			maxRate: 0.01,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := tt.build(tt.cfg)
			if err != nil {
				t.Fatal(err)
			}
			insertN := int(tt.cfg.ExpectedCapacity)
			if insertN == 0 {
				insertN = n
			}
			assertEmpiricalFPR(t, f, insertN, trials, tt.maxRate)
		})
	}
}

func TestEmpiricalFPRProperty(t *testing.T) {
	prop := func(capacity uint16, targetMilli uint16) bool {
		if capacity < 100 {
			capacity = 100
		}
		if capacity > 2000 {
			capacity = 2000
		}
		if targetMilli < 10 {
			targetMilli = 10
		}
		if targetMilli > 500 {
			targetMilli = 500
		}
		targetP := float64(targetMilli) / 10000
		n := int(capacity)
		if n > 1500 {
			n = 1500
		}
		trials := 1000

		cfg := TargetConfig(uint64(n), targetP)
		theory, err := cfg.TheoryFPRAt(uint64(n))
		if err != nil {
			return false
		}
		maxRate := math.Max(targetP*8, theory*4)
		if maxRate < 0.05 {
			maxRate = 0.05
		}

		f, err := newStandardEmpiricalFilter(cfg)
		if err != nil {
			return false
		}
		rate, err := measureEmpiricalFPR(f, n, trials)
		if err != nil {
			return false
		}
		return rate <= maxRate
	}
	cfg := &quick.Config{MaxCount: 50}
	if err := quick.Check(prop, cfg); err != nil {
		t.Error(err)
	}
}

// TestFalsePositiveRatePerStrategy delegates to the shared empirical harness.
func TestFalsePositiveRatePerStrategy(t *testing.T) {
	const n = 5000
	const trials = 5000

	for _, strategy := range AllStrategies() {
		t.Run(strategy.String(), func(t *testing.T) {
			cfg := TargetConfig(n, 0.01, WithHash(strategy))
			f, err := newStandardEmpiricalFilter(cfg)
			if err != nil {
				t.Fatal(err)
			}
			assertEmpiricalFPR(t, f, n, trials, 0.05)
		})
	}
}

func TestEmpiricalFPRReportsRate(t *testing.T) {
	cfg := TargetConfig(500, 0.05)
	f, err := newStandardEmpiricalFilter(cfg)
	if err != nil {
		t.Fatal(err)
	}
	rate, err := measureEmpiricalFPR(f, 500, 2000)
	if err != nil {
		t.Fatal(err)
	}
	if rate < 0 || rate > 1 {
		t.Fatalf("rate = %g, want in [0, 1]", rate)
	}
	t.Logf("empirical FPR at p=0.05: %.4f", rate)
}

func BenchmarkMeasureEmpiricalFPR(b *testing.B) {
	cfg := TargetConfig(1000, 0.01)
	f, err := newStandardEmpiricalFilter(cfg)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := measureEmpiricalFPR(f, 1000, 1000); err != nil {
			b.Fatal(err)
		}
	}
}
