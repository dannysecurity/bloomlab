package bloom

import "math"

// BucketSpread summarizes how uniformly double-hash positions land across m buckets.
// Use it to compare hash strategies or tune seeds before deploying a filter.
type BucketSpread struct {
	M            uint64
	K            uint
	Samples      int
	Probes       int
	MinCount     int
	MaxCount     int
	MeanCount    float64
	ChiSquared   float64
	EmptyBuckets int
}

// MeasureBucketSpread probes k bit positions per sample key and records how often
// each bucket index in [0, m) is hit. keyFor receives sample indices 0..samples-1.
func MeasureBucketSpread(h Hasher, m uint64, k uint, samples int, keyFor func(i int) []byte) BucketSpread {
	buckets := make([]int, m)
	probes := 0

	for i := 0; i < samples; i++ {
		key := keyFor(i)
		if len(key) == 0 {
			continue
		}
		h1, h2 := h.Derive(key)
		for j := uint(0); j < k; j++ {
			buckets[bitIndex(h1, h2, m, j)]++
			probes++
		}
	}

	minCount, maxCount := buckets[0], buckets[0]
	empty := 0
	var sum int
	for _, count := range buckets {
		sum += count
		if count < minCount {
			minCount = count
		}
		if count > maxCount {
			maxCount = count
		}
		if count == 0 {
			empty++
		}
	}

	mean := float64(sum) / float64(m)
	var chi float64
	if mean > 0 {
		for _, count := range buckets {
			diff := float64(count) - mean
			chi += diff * diff / mean
		}
	}

	return BucketSpread{
		M:            m,
		K:            k,
		Samples:      samples,
		Probes:       probes,
		MinCount:     minCount,
		MaxCount:     maxCount,
		MeanCount:    mean,
		ChiSquared:   chi,
		EmptyBuckets: empty,
	}
}

// WithinSpreadTolerance reports whether every bucket count lies within
// [mean/ratio, mean*ratio]. ratio must be positive.
func (s BucketSpread) WithinSpreadTolerance(ratio float64) bool {
	if ratio <= 0 || s.MeanCount == 0 {
		return false
	}
	low := s.MeanCount / ratio
	high := s.MeanCount * ratio
	return float64(s.MinCount) >= low && float64(s.MaxCount) <= high
}

// ChiSquaredBelow returns true when the uniformity score is below the given bound.
// Lower values indicate closer adherence to the uniform model.
func (s BucketSpread) ChiSquaredBelow(limit float64) bool {
	return s.ChiSquared < limit
}

// CompareBucketSpread ranks strategies by chi-squared uniformity (lower is better).
func CompareBucketSpread(m uint64, k uint, samples int, keyFor func(i int) []byte, strategies ...Strategy) []BucketSpread {
	out := make([]BucketSpread, 0, len(strategies))
	for _, strategy := range strategies {
		out = append(out, MeasureBucketSpread(NewHasher(strategy, 0), m, k, samples, keyFor))
	}
	return out
}

// ProbeOverlap summarizes duplicate bit probes within a single key's k hash positions.
// Lower overlap rates indicate better use of the k probes under double hashing.
type ProbeOverlap struct {
	Samples         int
	K               uint
	TotalProbes     int
	DuplicateProbes int
	OverlapRate     float64
}

// MeasureProbeOverlap counts how often double-hash iterations for the same key
// land on a bit index already probed for that key.
func MeasureProbeOverlap(h Hasher, m uint64, k uint, samples int, keyFor func(i int) []byte) ProbeOverlap {
	if samples <= 0 || k == 0 || m == 0 || keyFor == nil {
		return ProbeOverlap{K: k}
	}

	total := 0
	dupes := 0
	for i := 0; i < samples; i++ {
		key := keyFor(i)
		if len(key) == 0 {
			continue
		}
		seen := make(map[uint64]struct{}, k)
		h1, h2 := h.Derive(key)
		for j := uint(0); j < k; j++ {
			idx := bitIndex(h1, h2, m, j)
			total++
			if _, ok := seen[idx]; ok {
				dupes++
			} else {
				seen[idx] = struct{}{}
			}
		}
	}

	rate := 0.0
	if total > 0 {
		rate = float64(dupes) / float64(total)
	}
	return ProbeOverlap{
		Samples:         samples,
		K:               k,
		TotalProbes:     total,
		DuplicateProbes: dupes,
		OverlapRate:     rate,
	}
}

// BestUniformStrategy picks the strategy with the lowest chi-squared score.
func BestUniformStrategy(m uint64, k uint, samples int, keyFor func(i int) []byte, strategies []Strategy) Strategy {
	if len(strategies) == 0 {
		return HashFNV
	}
	best := strategies[0]
	bestChi := math.MaxFloat64
	for _, strategy := range strategies {
		spread := MeasureBucketSpread(NewHasher(strategy, 0), m, k, samples, keyFor)
		if spread.ChiSquared < bestChi {
			bestChi = spread.ChiSquared
			best = strategy
		}
	}
	return best
}
