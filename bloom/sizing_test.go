package bloom

import (
	"math"
	"testing"
	"testing/quick"
)

func TestOptimalSizingGoldenValues(t *testing.T) {
	tests := []struct {
		name string
		n    uint64
		p    float64
		wantM uint64
		wantK uint
	}{
		{"10k at 1%", 10_000, 0.01, 95850, 6},
		{"1k at 0.1%", 1_000, 0.001, 14377, 9},
		{"5k at 1%", 5_000, 0.01, 47925, 6},
		{"100 at 10%", 100, 0.1, 479, 3},
		{"min bits floor", 10, 0.5, 64, 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := TargetConfig(tt.n, tt.p)
			m, k, err := cfg.Size()
			if err != nil {
				t.Fatal(err)
			}
			if m != tt.wantM {
				t.Errorf("m = %d, want %d", m, tt.wantM)
			}
			if k != tt.wantK {
				t.Errorf("k = %d, want %d", k, tt.wantK)
			}
		})
	}
}

func TestOptimalSizingBounds(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		minM    uint64
		minK    uint
		maxK    uint
	}{
		{
			name: "default bounds",
			cfg:  TargetConfig(5000, 0.01),
			minM: defaultMinBits,
			minK: 1,
			maxK: defaultMaxHashCount,
		},
		{
			name: "custom min bits",
			cfg:  TargetConfig(50, 0.25, WithMinBits(256)),
			minM: 256,
			minK: 1,
			maxK: defaultMaxHashCount,
		},
		{
			name: "max hash cap",
			cfg:  TargetConfig(200_000, 0.0001, WithMaxHashCount(10)),
			minM: defaultMinBits,
			minK: 1,
			maxK: 10,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, k, err := tt.cfg.Size()
			if err != nil {
				t.Fatal(err)
			}
			if m < tt.minM {
				t.Errorf("m = %d, want >= %d", m, tt.minM)
			}
			if k < tt.minK {
				t.Errorf("k = %d, want >= %d", k, tt.minK)
			}
			if k > tt.maxK {
				t.Errorf("k = %d, want <= %d", k, tt.maxK)
			}
		})
	}
}

func TestOptimalSizingProperty(t *testing.T) {
	prop := func(capacity uint32, targetMilli uint16) bool {
		if capacity == 0 || capacity > 100_000 {
			return true
		}
		if targetMilli < 1 {
			targetMilli = 1
		}
		if targetMilli >= 9999 {
			targetMilli = 9999
		}
		p := float64(targetMilli) / 10000
		cfg := TargetConfig(uint64(capacity), p)
		m, k, err := cfg.Size()
		if err != nil {
			return false
		}
		if m < defaultMinBits {
			return false
		}
		if k < 1 || k > defaultMaxHashCount {
			return false
		}
		theory, err := cfg.TheoryFPRAt(uint64(capacity))
		if err != nil {
			return false
		}
		// Integer truncation of m and k can push theory slightly above target.
		return theory <= p*1.2
	}
	cfg := &quick.Config{MaxCount: 200}
	if err := quick.Check(prop, cfg); err != nil {
		t.Error(err)
	}
}

func TestOptimalKMonotonicInM(t *testing.T) {
	const n = 10_000
	tests := []struct {
		name string
		p    float64
	}{
		{"1% fpr", 0.01},
		{"0.1% fpr", 0.001},
		{"10% fpr", 0.1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := TargetConfig(n, tt.p)
			m, k, err := cfg.Size()
			if err != nil {
				t.Fatal(err)
			}
			prevK := k
			for extra := uint64(0); extra <= m/2; extra += m / 8 {
				largerM := m + extra
				if largerM == m {
					continue
				}
				nextK := optimalK(largerM, n, defaultMaxHashCount)
				if nextK < prevK {
					t.Errorf("k decreased from %d to %d when m grew %d -> %d", prevK, nextK, m, largerM)
				}
				prevK = nextK
			}
		})
	}
}

func TestTheoryFalsePositiveRateGolden(t *testing.T) {
	tests := []struct {
		name string
		n    uint64
		m    uint64
		k    uint
		want float64
	}{
		{
			name: "empty filter",
			n:    0,
			m:    1000,
			k:    4,
			want: 0,
		},
		{
			name: "10k capacity optimal sizing",
			n:    10_000,
			m:    95850,
			k:    6,
			want: 0.00998,
		},
		{
			name: "single hash function",
			n:    100,
			m:    1000,
			k:    1,
			want: 0.0952,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TheoryFalsePositiveRate(tt.n, tt.m, tt.k)
			if tt.want == 0 {
				if got != 0 {
					t.Errorf("TheoryFalsePositiveRate() = %g, want 0", got)
				}
				return
			}
			if math.Abs(got-tt.want) > 0.001 {
				t.Errorf("TheoryFalsePositiveRate() = %g, want ~%g", got, tt.want)
			}
		})
	}
}
