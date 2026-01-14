package bloom

import "math"

// DoubleHashStride summarizes how well h2 paces double-hash probes across m buckets.
// When gcd(h2, m) > 1 the probe sequence visits fewer than m distinct positions.
type DoubleHashStride struct {
	M              uint64
	K              uint
	Samples        int
	GCDgtOne       int
	GCDgtOneRate   float64
	ShortCycleKeys int
	MinReachable   int
	MaxReachable   int
	MeanReachable  float64
}

// MeasureDoubleHashStride evaluates stride quality for a hasher at layout (m, k).
func MeasureDoubleHashStride(h Hasher, m uint64, k uint, samples int, keyFor func(i int) []byte) DoubleHashStride {
	if samples <= 0 || m == 0 || k == 0 || keyFor == nil {
		return DoubleHashStride{M: m, K: k}
	}

	want := int(k)
	if uint64(want) > m {
		want = int(m)
	}

	gcdBad := 0
	short := 0
	minReach, maxReach := want, 0
	var sumReach float64
	seenSamples := 0

	for i := 0; i < samples; i++ {
		key := keyFor(i)
		if len(key) == 0 {
			continue
		}
		h1, h2 := h.Derive(key)
		_ = h1
		g := gcdUint64(h2, m)
		if g > 1 {
			gcdBad++
		}

		reachable := reachableProbeCount(h1, h2, m, k)
		maxCycle := int(m / g)
		if maxCycle > want {
			maxCycle = want
		}
		if g > 1 && reachable < maxCycle {
			short++
		}
		if reachable < minReach {
			minReach = reachable
		}
		if reachable > maxReach {
			maxReach = reachable
		}
		sumReach += float64(reachable)
		seenSamples++
	}

	rate := 0.0
	mean := 0.0
	if seenSamples > 0 {
		rate = float64(gcdBad) / float64(seenSamples)
		mean = sumReach / float64(seenSamples)
	}

	return DoubleHashStride{
		M:              m,
		K:              k,
		Samples:        seenSamples,
		GCDgtOne:       gcdBad,
		GCDgtOneRate:   rate,
		ShortCycleKeys: short,
		MinReachable:   minReach,
		MaxReachable:   maxReach,
		MeanReachable:  mean,
	}
}

// StrideHealthy reports whether gcd(h2, m) defects are within acceptable bounds.
func (s DoubleHashStride) StrideHealthy(maxGCDRate float64) bool {
	if s.Samples == 0 {
		return false
	}
	return s.GCDgtOneRate <= maxGCDRate
}

// H1H2Correlation measures linear correlation between derived h1 and h2 values.
// Values near zero indicate independent seeds suitable for double hashing.
type H1H2Correlation struct {
	Samples int
	Pearson float64
}

// MeasureH1H2Correlation computes the Pearson correlation of (h1, h2) over sample keys.
func MeasureH1H2Correlation(h Hasher, samples int, keyFor func(i int) []byte) H1H2Correlation {
	if samples <= 0 || keyFor == nil {
		return H1H2Correlation{}
	}

	var sumX, sumY, sumXX, sumYY, sumXY float64
	n := 0

	for i := 0; i < samples; i++ {
		key := keyFor(i)
		if len(key) == 0 {
			continue
		}
		h1, h2 := h.Derive(key)
		x := float64(h1)
		y := float64(h2)
		sumX += x
		sumY += y
		sumXX += x * x
		sumYY += y * y
		sumXY += x * y
		n++
	}

	if n == 0 {
		return H1H2Correlation{}
	}

	fn := float64(n)
	num := fn*sumXY - sumX*sumY
	denX := fn*sumXX - sumX*sumX
	denY := fn*sumYY - sumY*sumY
	r := 0.0
	if denX > 0 && denY > 0 {
		r = num / math.Sqrt(denX*denY)
	}

	return H1H2Correlation{Samples: n, Pearson: r}
}

// CorrelationBelow returns true when |Pearson| is below the given bound.
func (c H1H2Correlation) CorrelationBelow(limit float64) bool {
	if c.Samples == 0 {
		return false
	}
	return math.Abs(c.Pearson) < limit
}

func reachableProbeCount(h1, h2, m uint64, k uint) int {
	seen := make(map[uint64]struct{}, k)
	for i := uint(0); i < k; i++ {
		seen[bitIndex(h1, h2, m, i)] = struct{}{}
	}
	return len(seen)
}

func gcdUint64(a, b uint64) uint64 {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}
