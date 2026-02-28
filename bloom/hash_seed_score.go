package bloom

import "math"

// SeedQualityScore combines bucket spread, probe overlap, double-hash stride,
// and h1/h2 correlation into a single ranking metric. Lower is better.
// Penalty terms scale with chi-squared so they stay comparable across layouts.
func SeedQualityScore(spread BucketSpread, overlap ProbeOverlap, stride DoubleHashStride, corr H1H2Correlation) float64 {
	chi := spread.ChiSquared
	if chi <= 0 {
		chi = 1
	}
	overlapPenalty := overlap.OverlapRate * chi * 2
	gcdPenalty := stride.GCDgtOneRate * chi * 3
	corrPenalty := math.Abs(corr.Pearson) * chi
	return chi + overlapPenalty + gcdPenalty + corrPenalty
}

func evaluateSeedCandidate(strategy Strategy, seed uint64, opts TuneOptions) SeedCandidate {
	hasher := NewHasher(strategy, seed)
	spread := MeasureBucketSpread(hasher, opts.M, opts.K, opts.Samples, opts.KeyFor)
	overlap := MeasureProbeOverlap(hasher, opts.M, opts.K, opts.Samples, opts.KeyFor)
	stride := MeasureDoubleHashStride(hasher, opts.M, opts.K, opts.Samples, opts.KeyFor)
	corr := MeasureH1H2Correlation(hasher, opts.Samples, opts.KeyFor)
	score := SeedQualityScore(spread, overlap, stride, corr)
	return SeedCandidate{
		Seed:        seed,
		Spread:      spread,
		Overlap:     overlap,
		Stride:      stride,
		Correlation: corr,
		ChiSquared:  spread.ChiSquared,
		Score:       score,
	}
}
