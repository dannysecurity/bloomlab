package bloom

import (
	"math"
	"testing"
)

func TestTheoryFalsePositiveRate(t *testing.T) {
	tests := []struct {
		name string
		n    uint64
		m    uint64
		k    uint
		want float64
	}{
		{"zero inserts", 0, 1000, 4, 0},
		{"zero bits", 100, 0, 4, 0},
		{"zero hash functions", 100, 1000, 0, 0},
		{
			name: "half fill optimal k",
			n:    1000,
			m:    10000,
			k:    uint(math.Round(float64(10000) / 1000 * math.Ln2)),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TheoryFalsePositiveRate(tt.n, tt.m, tt.k)
			if tt.want != 0 && math.Abs(got-tt.want) > 1e-9 {
				t.Errorf("TheoryFalsePositiveRate() = %g, want %g", got, tt.want)
			}
			if tt.n == 0 || tt.m == 0 || tt.k == 0 {
				if got != 0 {
					t.Errorf("TheoryFalsePositiveRate() = %g, want 0", got)
				}
			}
		})
	}
}

func TestTargetConfigTheoryFPR(t *testing.T) {
	tests := []struct {
		n       uint64
		target  float64
		maxRate float64
	}{
		{10_000, 0.01, 0.012},
		{1_000, 0.001, 0.0015},
		{100, 0.1, 0.12},
	}
	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			cfg := TargetConfig(tt.n, tt.target)
			p, err := cfg.TheoryFPRAt(tt.n)
			if err != nil {
				t.Fatal(err)
			}
			if p > tt.maxRate {
				t.Errorf("TheoryFPRAt(%d) = %g, want <= %g", tt.n, p, tt.maxRate)
			}
			if p < tt.target*0.5 {
				t.Errorf("TheoryFPRAt(%d) = %g, unexpectedly far below target %g", tt.n, p, tt.target)
			}
		})
	}
}

func TestFilterTheoryFPR(t *testing.T) {
	const n = 1000
	f, err := New(n, 0.01)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < n; i++ {
		f.Add([]byte{byte(i >> 8), byte(i)})
	}

	atCapacity := f.TheoryFPR()
	if atCapacity > 0.012 {
		t.Errorf("TheoryFPR at capacity = %g, want ~0.01", atCapacity)
	}

	f.Add([]byte("extra"))
	afterExtra := f.TheoryFPR()
	if afterExtra <= atCapacity {
		t.Errorf("TheoryFPR after extra insert = %g, want > at-capacity %g", afterExtra, atCapacity)
	}
}
