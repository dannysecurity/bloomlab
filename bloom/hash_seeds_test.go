package bloom

import "testing"

func TestSplitMix64Deterministic(t *testing.T) {
	a := splitMix64(42)
	b := splitMix64(42)
	if a != b {
		t.Fatalf("splitMix64(42) = %#x vs %#x", a, b)
	}
	if a == 42 {
		t.Fatal("splitMix64 should mix the input")
	}
}

func TestExpandTuneSeedsPreservesBase(t *testing.T) {
	base := []uint64{0, 42}
	expanded := ExpandTuneSeeds(base, 3)
	if len(expanded) < len(base) {
		t.Fatalf("got %d seeds, want at least %d", len(expanded), len(base))
	}
	seen := make(map[uint64]struct{}, len(expanded))
	for _, seed := range expanded {
		seen[seed] = struct{}{}
	}
	for _, seed := range base {
		if _, ok := seen[seed]; !ok {
			t.Fatalf("missing base seed %d in expanded ladder", seed)
		}
	}
}

func TestExpandTuneSeedsDeduplicates(t *testing.T) {
	// Two bases that collapse to the same neighbor after one SplitMix step.
	base := []uint64{0, 1}
	expanded := ExpandTuneSeeds(base, 5)
	seen := make(map[uint64]struct{}, len(expanded))
	for _, seed := range expanded {
		if _, ok := seen[seed]; ok {
			t.Fatalf("duplicate seed %d in expanded ladder", seed)
		}
		seen[seed] = struct{}{}
	}
}

func TestExpandTuneSeedsZeroSteps(t *testing.T) {
	base := []uint64{7, 99}
	got := ExpandTuneSeeds(base, 0)
	if len(got) != len(base) {
		t.Fatalf("got %d seeds, want %d", len(got), len(base))
	}
	for i := range base {
		if got[i] != base[i] {
			t.Fatalf("got[%d] = %d, want %d", i, got[i], base[i])
		}
	}
}

func TestResolveTuneSeedsDefaultsAndExpands(t *testing.T) {
	defaults := ResolveTuneSeeds(nil, 0)
	if len(defaults) != len(DefaultTuneSeeds()) {
		t.Fatalf("got %d default seeds, want %d", len(defaults), len(DefaultTuneSeeds()))
	}

	expanded := ResolveTuneSeeds([]uint64{42}, 2)
	if len(expanded) != 3 {
		t.Fatalf("got %d expanded seeds, want 3", len(expanded))
	}
	if expanded[0] != 42 {
		t.Fatalf("first seed = %d, want 42", expanded[0])
	}
}

func TestRecommendHasherExpandSeeds(t *testing.T) {
	keyFor := func(i int) []byte {
		return []byte{byte(i), byte(i >> 8)}
	}
	base := TuneOptions{M: 1024, K: 4, Samples: 2000, KeyFor: keyFor}
	narrow := RecommendHasher(base, []Strategy{HashMurmur3}, []uint64{0, 7})
	expanded := base
	expanded.ExpandSeeds = 2
	wide := RecommendHasher(expanded, []Strategy{HashMurmur3}, []uint64{0, 7})
	if len(narrow.Candidates) >= len(wide.Candidates) {
		t.Fatalf("expand-seeds expected more candidates: narrow=%d wide=%d",
			len(narrow.Candidates), len(wide.Candidates))
	}
}
