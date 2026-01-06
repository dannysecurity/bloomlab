package bloom

import (
	"fmt"
	"math"
	"strings"
)

// ContinuousOptimalM returns the real-valued bit count from the standard Bloom
// sizing formula before truncation or MinBits floors.
func ContinuousOptimalM(n uint64, p float64) float64 {
	return -float64(n) * math.Log(p) / (math.Ln2 * math.Ln2)
}

// ContinuousOptimalK returns the real-valued hash-function count at the
// continuous optimum before rounding or MaxHashCount caps.
func ContinuousOptimalK(m, n uint64) float64 {
	if n == 0 {
		return 0
	}
	return float64(m) / float64(n) * math.Ln2
}

// FormatSizingDerivation renders numbered false-positive math steps for a
// sizing plan. It mirrors the README derivation and is used by fprcalc -derive.
func FormatSizingDerivation(p SizingPlan) string {
	n := p.ExpectedCapacity
	target := p.TargetFPR
	mCont := ContinuousOptimalM(n, target)
	m := p.Bits
	kCont := ContinuousOptimalK(m, n)
	k := p.HashCount
	knOverM := float64(k) * float64(n) / float64(m)
	clearProb := math.Exp(-knOverM)
	fill := p.FillFraction
	achieved := p.AchievedFPR

	var b strings.Builder
	fmt.Fprintf(&b, "False-positive math for n=%d, target p=%g\n\n", n, target)

	fmt.Fprintf(&b, "1. Optimal bits (continuous)\n")
	fmt.Fprintf(&b, "   m = -n·ln(p) / (ln 2)²\n")
	fmt.Fprintf(&b, "   m = -%d·ln(%g) / (ln 2)² ≈ %.2f", n, target, mCont)
	if m == uint64(mCont) {
		fmt.Fprintf(&b, " → %d bits\n\n", m)
	} else {
		fmt.Fprintf(&b, " → %d bits after sizing bounds\n\n", m)
	}

	fmt.Fprintf(&b, "2. Hash functions at the continuous optimum\n")
	fmt.Fprintf(&b, "   k* = (m/n)·ln 2\n")
	fmt.Fprintf(&b, "   k* = (%d/%d)·ln 2 ≈ %.3f → k=%d\n\n", m, n, kCont, k)

	fmt.Fprintf(&b, "3. Fill fraction after n distinct inserts\n")
	fmt.Fprintf(&b, "   f = 1 - e^(-kn/m)\n")
	fmt.Fprintf(&b, "   kn/m = %d·%d/%d ≈ %.4f\n", k, n, m, knOverM)
	fmt.Fprintf(&b, "   P(clear) ≈ e^(-kn/m) ≈ %.4f\n", clearProb)
	fmt.Fprintf(&b, "   f ≈ 1 - e^(-kn/m) ≈ %.4f (%.1f%%)\n\n", fill, fill*100)

	fmt.Fprintf(&b, "4. False positive on an absent key\n")
	fmt.Fprintf(&b, "   p ≈ f^k ≈ (1 - e^(-kn/m))^k\n")
	fmt.Fprintf(&b, "   p ≈ %.4f^%d ≈ %.5f (%.3f%%)", fill, k, achieved, achieved*100)
	if achieved <= target {
		fmt.Fprintf(&b, " — at or below target p=%g\n", target)
	} else {
		fmt.Fprintf(&b, " — slightly above target p=%g (integer rounding)\n", target)
	}

	fmt.Fprintf(&b, "\n5. Continuous optimum shortcut (before rounding)\n")
	fmt.Fprintf(&b, "   At k* = (m/n)·ln 2 we have kn/m = ln 2, so f ≈ 1/2 and\n")
	fmt.Fprintf(&b, "   p ≈ (1/2)^k* = 2^(-(m/n)·ln 2) = e^(-(m/n)·(ln 2)²).\n")
	fmt.Fprintf(&b, "   Solving for m gives m = -n·ln(p) / (ln 2)² (step 1).\n")

	return b.String()
}
