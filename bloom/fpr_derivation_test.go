package bloom

import (
	"fmt"
	"math"
	"strings"
	"testing"
)

func TestContinuousOptimalM(t *testing.T) {
	got := ContinuousOptimalM(10_000, 0.01)
	want := 95850.35
	if math.Abs(got-want) > 1 {
		t.Errorf("ContinuousOptimalM() = %g, want ~%g", got, want)
	}
}

func TestContinuousOptimalK(t *testing.T) {
	got := ContinuousOptimalK(95850, 10_000)
	want := 6.64
	if math.Abs(got-want) > 0.01 {
		t.Errorf("ContinuousOptimalK() = %g, want ~%g", got, want)
	}
}

func TestFormatSizingDerivationGolden(t *testing.T) {
	plan, err := PlanSizing(10_000, 0.01)
	if err != nil {
		t.Fatal(err)
	}
	text := FormatSizingDerivation(plan)
	for _, want := range []string{
		"False-positive math for n=10000, target p=0.01",
		"m = -n·ln(p) / (ln 2)²",
		"k* = (m/n)·ln 2",
		"f = 1 - e^(-kn/m)",
		"p ≈ f^k",
		"95850",
		"k=6",
		"Continuous optimum shortcut",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("FormatSizingDerivation() missing %q:\n%s", want, text)
		}
	}
}

func TestFormatSizingDerivationMatchesPlan(t *testing.T) {
	plan, err := PlanSizing(1000, 0.001)
	if err != nil {
		t.Fatal(err)
	}
	text := FormatSizingDerivation(plan)
	if !strings.Contains(text, fmt.Sprintf("≈ %.5f", plan.AchievedFPR)) {
		t.Fatalf("derivation should echo achieved FPR %.5f:\n%s", plan.AchievedFPR, text)
	}
}
