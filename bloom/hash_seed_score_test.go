package bloom

import "testing"

func TestSeedQualityScorePenalties(t *testing.T) {
	base := BucketSpread{ChiSquared: 100}
	overlap := ProbeOverlap{OverlapRate: 0}
	stride := DoubleHashStride{GCDgtOneRate: 0}
	corr := H1H2Correlation{Pearson: 0}

	baseScore := SeedQualityScore(base, overlap, stride, corr)
	if baseScore != 100 {
		t.Fatalf("base score = %f, want 100", baseScore)
	}

	overlapScore := SeedQualityScore(base, ProbeOverlap{OverlapRate: 0.1}, stride, corr)
	if overlapScore <= baseScore {
		t.Fatalf("overlap penalty expected score > %f, got %f", baseScore, overlapScore)
	}

	gcdScore := SeedQualityScore(base, overlap, DoubleHashStride{GCDgtOneRate: 0.2}, corr)
	if gcdScore <= baseScore {
		t.Fatalf("gcd penalty expected score > %f, got %f", baseScore, gcdScore)
	}

	corrScore := SeedQualityScore(base, overlap, stride, H1H2Correlation{Pearson: 0.5})
	if corrScore <= baseScore {
		t.Fatalf("correlation penalty expected score > %f, got %f", baseScore, corrScore)
	}
}

func TestSeedQualityScoreZeroChiUsesFloor(t *testing.T) {
	score := SeedQualityScore(
		BucketSpread{ChiSquared: 0},
		ProbeOverlap{OverlapRate: 0.1},
		DoubleHashStride{},
		H1H2Correlation{},
	)
	if score <= 1 {
		t.Fatalf("expected penalties on chi floor, got %f", score)
	}
}

func TestEvaluateSeedCandidatePopulatesMetrics(t *testing.T) {
	opts := TuneOptions{
		M:       512,
		K:       4,
		Samples: 500,
		KeyFor: func(i int) []byte {
			return []byte{byte(i), byte(i >> 8)}
		},
	}
	candidate := evaluateSeedCandidate(HashMurmur3, 42, opts)
	if candidate.Score <= 0 {
		t.Fatalf("expected positive score, got %f", candidate.Score)
	}
	if candidate.ChiSquared != candidate.Spread.ChiSquared {
		t.Fatalf("chi² mismatch: candidate=%f spread=%f", candidate.ChiSquared, candidate.Spread.ChiSquared)
	}
	if candidate.Overlap.TotalProbes == 0 {
		t.Fatal("expected overlap measurement")
	}
	if candidate.Stride.Samples == 0 {
		t.Fatal("expected stride measurement")
	}
	if candidate.Correlation.Samples == 0 {
		t.Fatal("expected correlation measurement")
	}
}
