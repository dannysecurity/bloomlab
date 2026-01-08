package bloom

import (
	"math"
	"strings"
	"testing"
)

func TestTheoryFillFraction(t *testing.T) {
	tests := []struct {
		name string
		n    uint64
		m    uint64
		k    uint
		want float64
	}{
		{"zero inserts", 0, 1000, 4, 0},
		{"half fill optimal k", 1000, 10000, uint(math.Round(float64(10000)/1000*math.Ln2)), 0.503},
		{"10k capacity optimal sizing", 10_000, 95850, 6, 0.465},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TheoryFillFraction(tt.n, tt.m, tt.k)
			if tt.want == 0 {
				if got != 0 {
					t.Errorf("TheoryFillFraction() = %g, want 0", got)
				}
				return
			}
			if math.Abs(got-tt.want) > 0.01 {
				t.Errorf("TheoryFillFraction() = %g, want ~%g", got, tt.want)
			}
		})
	}
}

func TestTheoryFillFractionMatchesFPRBase(t *testing.T) {
	const n = 5000
	cfg := TargetConfig(n, 0.01)
	m, k, err := cfg.Size()
	if err != nil {
		t.Fatal(err)
	}
	fill := TheoryFillFraction(n, m, k)
	fpr := TheoryFalsePositiveRate(n, m, k)
	if math.Abs(fpr-math.Pow(fill, float64(k))) > 1e-12 {
		t.Errorf("FPR %g != fill^k %g", fpr, math.Pow(fill, float64(k)))
	}
}

func TestPlanSizingGolden(t *testing.T) {
	plan, err := PlanSizing(10_000, 0.01)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Bits != 95850 {
		t.Errorf("Bits = %d, want 95850", plan.Bits)
	}
	if plan.HashCount != 6 {
		t.Errorf("HashCount = %d, want 6", plan.HashCount)
	}
	if plan.AchievedFPR > 0.0102 {
		t.Errorf("AchievedFPR = %g, want <= ~0.0102 (slightly above target due to integer rounding)", plan.AchievedFPR)
	}
	if math.Abs(plan.AchievedFPR-0.01014) > 0.001 {
		t.Errorf("AchievedFPR = %g, want ~0.01014", plan.AchievedFPR)
	}
	if math.Abs(plan.FillFraction-0.465) > 0.01 {
		t.Errorf("FillFraction = %g, want ~0.465", plan.FillFraction)
	}
}

func TestPlanSizingString(t *testing.T) {
	plan, err := PlanSizing(1000, 0.01)
	if err != nil {
		t.Fatal(err)
	}
	s := plan.String()
	for _, want := range []string{"target n=1000", "p=0.01", "bits/item", "theory FPR"} {
		if !strings.Contains(s, want) {
			t.Errorf("String() = %q, want substring %q", s, want)
		}
	}
}

func TestPlanSizingFromRespectsBounds(t *testing.T) {
	cfg := TargetConfig(100_000, 0.001, WithMinBits(256), WithMaxHashCount(8))
	plan, err := PlanSizingFrom(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Bits < 256 {
		t.Errorf("Bits = %d, want >= 256", plan.Bits)
	}
	if plan.HashCount > 8 {
		t.Errorf("HashCount = %d, want <= 8", plan.HashCount)
	}

	direct, err := PlanSizing(100_000, 0.001, WithMinBits(256), WithMaxHashCount(8))
	if err != nil {
		t.Fatal(err)
	}
	if plan != direct {
		t.Fatalf("PlanSizingFrom = %+v, PlanSizing = %+v", plan, direct)
	}
}

func TestPlanSizingFromRejectsExplicitConfig(t *testing.T) {
	if _, err := PlanSizingFrom(ExplicitConfig(128, 4)); err != ErrInvalidCapacity {
		t.Fatalf("PlanSizingFrom(explicit) = %v, want ErrInvalidCapacity", err)
	}
}
