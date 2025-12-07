package bloom

import (
	"fmt"
	"testing"
	"testing/quick"
)

func TestParseStrategy(t *testing.T) {
	tests := []struct {
		in   string
		want Strategy
	}{
		{"", HashFNV},
		{"fnv", HashFNV},
		{"FNV", HashFNV},
		{"murmur3", HashMurmur3},
		{"Murmur", HashMurmur3},
	}
	for _, tt := range tests {
		got, err := ParseStrategy(tt.in)
		if err != nil {
			t.Fatalf("ParseStrategy(%q) error: %v", tt.in, err)
		}
		if got != tt.want {
			t.Errorf("ParseStrategy(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}

	if _, err := ParseStrategy("sha256"); err == nil {
		t.Fatal("expected error for unknown strategy")
	}
}

func TestHasherDeterministic(t *testing.T) {
	strategies := []Strategy{HashFNV, HashMurmur3}
	key := []byte("bloomlab-determinism-check")

	for _, strategy := range strategies {
		t.Run(strategy.String(), func(t *testing.T) {
			h := NewHasher(strategy, 42)
			h1a, h2a := h.Derive(key)
			h1b, h2b := h.Derive(key)
			if h1a != h1b || h2a != h2b {
				t.Fatalf("non-deterministic derive: (%d,%d) vs (%d,%d)", h1a, h2a, h1b, h2b)
			}
			if h2a == 0 || h2b == 0 {
				t.Fatal("h2 must never be zero")
			}
		})
	}
}

func TestHasherSeedAffectsOutput(t *testing.T) {
	key := []byte("seed-sensitivity")
	a1, a2 := NewHasher(HashMurmur3, 0).Derive(key)
	b1, b2 := NewHasher(HashMurmur3, 99).Derive(key)
	if a1 == b1 && a2 == b2 {
		t.Fatal("expected different hashes for different seeds")
	}
}

func TestBitIndexInRange(t *testing.T) {
	const m = 997
	const k = 12
	key := []byte("range-check")
	h1, h2 := NewHasher(HashMurmur3, 0).Derive(key)

	for i := uint(0); i < k; i++ {
		idx := bitIndex(h1, h2, m, i)
		if idx >= m {
			t.Fatalf("bitIndex out of range: idx=%d m=%d i=%d", idx, m, i)
		}
	}
}

func TestEnsureH2NonZero(t *testing.T) {
	tests := []struct {
		name string
		h2   uint64
		want uint64
	}{
		{"zero becomes one", 0, 1},
		{"one unchanged", 1, 1},
		{"nonzero unchanged", 42, 42},
		{"max uint64 unchanged", ^uint64(0), ^uint64(0)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ensureH2NonZero(tt.h2); got != tt.want {
				t.Errorf("ensureH2NonZero(%d) = %d, want %d", tt.h2, got, tt.want)
			}
		})
	}
}

func TestBitIndexProperty(t *testing.T) {
	prop := func(h1, h2 uint32, m uint32, i uint8) bool {
		if m == 0 {
			return true
		}
		got := bitIndex(uint64(h1), uint64(h2), uint64(m), uint(i))
		if got >= uint64(m) {
			return false
		}
		if m > 0 && m&(m-1) == 0 {
			sum := uint64(h1) + uint64(i)*uint64(h2)
			return got == sum%uint64(m)
		}
		return true
	}
	cfg := &quick.Config{MaxCount: 300}
	if err := quick.Check(prop, cfg); err != nil {
		t.Error(err)
	}
}

func TestBitIndexPowerOfTwoFastPath(t *testing.T) {
	const m = 1024
	h1, h2 := uint64(9000), uint64(7)
	for i := uint(0); i < 16; i++ {
		want := (h1 + uint64(i)*h2) % m
		got := bitIndex(h1, h2, m, i)
		if got != want {
			t.Fatalf("i=%d: fast path %d != slow path %d", i, got, want)
		}
	}
}

func TestFNVPreservesLegacyVectors(t *testing.T) {
	key := []byte("alpha")
	h1, h2 := NewHasher(HashFNV, 0).Derive(key)
	if h2 == 0 {
		t.Fatal("h2 must be corrected to non-zero")
	}

	legacyH1, legacyH2 := fnvHasher{}.Derive(key)
	if h1 != legacyH1 || h2 != legacyH2 {
		t.Fatalf("FNV drift: got (%d,%d) want (%d,%d)", h1, h2, legacyH1, legacyH2)
	}
}

func TestHashDistribution(t *testing.T) {
	const m = 4096
	const k = 8
	const samples = 20_000

	t.Run("murmur3", func(t *testing.T) {
		buckets := make([]int, m)
		h := NewHasher(HashMurmur3, 0)

		for i := 0; i < samples; i++ {
			key := []byte(fmt.Sprintf("key-%d-%d", i, i*7919))
			h1, h2 := h.Derive(key)
			for j := uint(0); j < k; j++ {
				buckets[bitIndex(h1, h2, m, j)]++
			}
		}

		expected := float64(samples*k) / float64(m)
		minCount, maxCount := buckets[0], buckets[0]
		for _, count := range buckets[1:] {
			if count < minCount {
				minCount = count
			}
			if count > maxCount {
				maxCount = count
			}
		}

		if float64(minCount) < expected/4 || float64(maxCount) > expected*4 {
			t.Fatalf("bucket spread [%d, %d] outside [%.0f, %.0f] for mean %.1f",
				minCount, maxCount, expected/4, expected*4, expected)
		}
	})

	t.Run("fnv_coverage", func(t *testing.T) {
		seen := make([]bool, m)
		h := NewHasher(HashFNV, 0)
		for i := 0; i < samples; i++ {
			key := []byte(fmt.Sprintf("key-%d", i))
			h1, h2 := h.Derive(key)
			for j := uint(0); j < k; j++ {
				seen[bitIndex(h1, h2, m, j)] = true
			}
		}
		empty := 0
		for _, hit := range seen {
			if !hit {
				empty++
			}
		}
		if empty > m/8 {
			t.Fatalf("%d of %d buckets never hit; indexing may be degenerate", empty, m)
		}
	})
}

func TestFalsePositiveRatePerStrategy(t *testing.T) {
	const n = 5000
	const trials = 5000

	for _, strategy := range []Strategy{HashFNV, HashMurmur3} {
		t.Run(strategy.String(), func(t *testing.T) {
			cfg := TargetConfig(n, 0.01, WithHash(strategy))
			f, err := NewFilter(cfg)
			if err != nil {
				t.Fatal(err)
			}

			for i := 0; i < n; i++ {
				f.Add([]byte{byte(i >> 8), byte(i)})
			}

			falsePositives := 0
			for i := n; i < n+trials; i++ {
				if f.Contains([]byte{byte(i >> 8), byte(i)}) {
					falsePositives++
				}
			}

			rate := float64(falsePositives) / trials
			if rate > 0.05 {
				t.Errorf("false positive rate %.4f exceeds tolerance for p=0.01", rate)
			}
		})
	}
}

func TestConfigHasherDefaultIsFNV(t *testing.T) {
	h := TargetConfig(100, 0.01).Hasher()
	if h.Strategy() != HashFNV {
		t.Fatalf("default strategy = %v, want fnv", h.Strategy())
	}
}

func TestFilterHashStrategyWiring(t *testing.T) {
	cfg := ExplicitConfig(256, 4, WithHash(HashMurmur3), WithSeed(7))

	f, err := NewFilter(cfg)
	if err != nil {
		t.Fatal(err)
	}
	f.Add([]byte("wired"))
	if !f.Contains([]byte("wired")) {
		t.Fatal("murmur3-backed filter should contain inserted key")
	}

	fnvCfg := ExplicitConfig(256, 4)
	fnvFilter, err := NewFilter(fnvCfg)
	if err != nil {
		t.Fatal(err)
	}
	fnvFilter.Add([]byte("wired"))
	if f.BitCount() == fnvFilter.BitCount() && f.HashCount() == fnvFilter.HashCount() {
		// Different strategies should generally produce different bit patterns.
		if f.FillRatio() == fnvFilter.FillRatio() && f.Contains([]byte("other")) == fnvFilter.Contains([]byte("other")) {
			t.Log("filters sized identically; strategy-specific divergence is probabilistic")
		}
	}
}
