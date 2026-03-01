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
		{"xxhash", HashXXHash},
		{"xxh64", HashXXHash},
		{"wyhash", HashWyhash},
		{"wy", HashWyhash},
		{"highway", HashHighway},
		{"hwy", HashHighway},
		{"siphash", HashSipHash},
		{"sip", HashSipHash},
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

func TestParseStrategyList(t *testing.T) {
	strategies, err := ParseStrategyList("fnv, murmur3, xxhash")
	if err != nil {
		t.Fatal(err)
	}
	if len(strategies) != 3 {
		t.Fatalf("got %d strategies, want 3", len(strategies))
	}
	if strategies[1] != HashMurmur3 {
		t.Fatalf("strategies[1] = %v, want murmur3", strategies[1])
	}
	if _, err := ParseStrategyList(""); err == nil {
		t.Fatal("expected error for empty list")
	}
}

func TestHasherDeterministic(t *testing.T) {
	key := []byte("bloomlab-determinism-check")

	for _, strategy := range AllStrategies() {
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
	for _, strategy := range []Strategy{HashMurmur3, HashXXHash, HashWyhash, HashHighway, HashSipHash} {
		t.Run(strategy.String(), func(t *testing.T) {
			a1, a2 := NewHasher(strategy, 0).Derive(key)
			b1, b2 := NewHasher(strategy, 99).Derive(key)
			if a1 == b1 && a2 == b2 {
				t.Fatal("expected different hashes for different seeds")
			}
		})
	}
}

func TestFNVSeedIgnored(t *testing.T) {
	key := []byte("fnv-seed-invariance")
	a1, a2 := NewHasher(HashFNV, 0).Derive(key)
	b1, b2 := NewHasher(HashFNV, 99).Derive(key)
	if a1 != b1 || a2 != b2 {
		t.Fatalf("FNV with seed=0 (%d,%d) should match seed=99 (%d,%d)", a1, a2, b1, b2)
	}
}

func TestMurmur3GoldenVectors(t *testing.T) {
	tests := []struct {
		key  string
		seed uint64
		h1   uint64
		h2   uint64
	}{
		{"", 0, 0x0, 0x1},
		{"alpha", 0, 0xffe53dd0983e1695, 0xd9bb04982603e41e},
		{"alpha", 42, 0xd10dbe75208d96f9, 0xd22261ef4c7cd250},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			h1, h2 := NewHasher(HashMurmur3, tt.seed).Derive([]byte(tt.key))
			if h1 != tt.h1 || h2 != tt.h2 {
				t.Fatalf("Derive(%q, %d) = (%#x, %#x), want (%#x, %#x)",
					tt.key, tt.seed, h1, h2, tt.h1, tt.h2)
			}
		})
	}
}

func TestPairDerivedSeeds(t *testing.T) {
	seed1, seed2 := pairDerivedSeeds(42)
	if seed1 != 42 {
		t.Fatalf("seed1 = %d, want 42", seed1)
	}
	if seed2 != 42^doubleHashSeedMix {
		t.Fatalf("seed2 = %#x, want %#x", seed2, 42^doubleHashSeedMix)
	}
	if seed1 == seed2 {
		t.Fatal("h1 and h2 seeds must differ")
	}
}

func TestXXHashGoldenVectors(t *testing.T) {
	tests := []struct {
		key  string
		seed uint64
		h1   uint64
		h2   uint64
	}{
		{"", 0, 0xef46db3751d8e999, 0xc4349fc93c010000},
		{"alpha", 0, 0xc758e1011dda5848, 0x7b65251ba528d50b},
		{"alpha", 42, 0x41f206d893836e6b, 0xd61f1732b38a81d6},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			h1, h2 := NewHasher(HashXXHash, tt.seed).Derive([]byte(tt.key))
			if h1 != tt.h1 || h2 != tt.h2 {
				t.Fatalf("Derive(%q, %d) = (%#x, %#x), want (%#x, %#x)",
					tt.key, tt.seed, h1, h2, tt.h1, tt.h2)
			}
		})
	}
}

func TestHighwayHashGoldenVectors(t *testing.T) {
	tests := []struct {
		key  string
		seed uint64
		h1   uint64
		h2   uint64
	}{
		{"", 0, 0x11fe85e1552efe32, 0x71cd38f67c4adabf},
		{"alpha", 0, 0xab934f46daf2d355, 0x7e9bddcbe34ff58c},
		{"alpha", 42, 0x29f8c3a574833d22, 0x8e8de65365fe687e},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			h1, h2 := NewHasher(HashHighway, tt.seed).Derive([]byte(tt.key))
			if h1 != tt.h1 || h2 != tt.h2 {
				t.Fatalf("Derive(%q, %d) = (%#x, %#x), want (%#x, %#x)",
					tt.key, tt.seed, h1, h2, tt.h1, tt.h2)
			}
		})
	}
}

func TestSipHashGoldenVectors(t *testing.T) {
	tests := []struct {
		key  string
		seed uint64
		h1   uint64
		h2   uint64
	}{
		{"", 0, 0xfa1df9926549a886, 0xc6daa92a49072279},
		{"alpha", 0, 0xa674498c991740a1, 0x12d94341820116a4},
		{"alpha", 42, 0xb6509b790450f9f9, 0x5bc8afa042a499e9},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			h1, h2 := NewHasher(HashSipHash, tt.seed).Derive([]byte(tt.key))
			if h1 != tt.h1 || h2 != tt.h2 {
				t.Fatalf("Derive(%q, %d) = (%#x, %#x), want (%#x, %#x)",
					tt.key, tt.seed, h1, h2, tt.h1, tt.h2)
			}
		})
	}
}

func TestWyhashGoldenVectors(t *testing.T) {
	tests := []struct {
		key  string
		seed uint64
		h1   uint64
		h2   uint64
	}{
		{"", 0, 0x0, 0x82eb3a1667734cee},
		{"alpha", 0, 0x20fa1908dd5af84e, 0xa9bc628f7e9a1129},
		{"alpha", 42, 0xe43ad249323c9032, 0xc8d46741cc844940},
	}
	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			h1, h2 := NewHasher(HashWyhash, tt.seed).Derive([]byte(tt.key))
			if h1 != tt.h1 || h2 != tt.h2 {
				t.Fatalf("Derive(%q, %d) = (%#x, %#x), want (%#x, %#x)",
					tt.key, tt.seed, h1, h2, tt.h1, tt.h2)
			}
		})
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

	keyFor := func(i int) []byte {
		return []byte(fmt.Sprintf("key-%d-%d", i, i*7919))
	}

	for _, strategy := range AllStrategies() {
		t.Run(strategy.String(), func(t *testing.T) {
			spread := MeasureBucketSpread(NewHasher(strategy, 0), m, k, samples, keyFor)
			if strategy == HashFNV {
				if spread.EmptyBuckets > m/8 {
					t.Fatalf("%d of %d buckets never hit; indexing may be degenerate", spread.EmptyBuckets, m)
				}
				return
			}
			if !spread.WithinSpreadTolerance(4) {
				t.Fatalf("bucket spread [%d, %d] outside 4x of mean %.1f",
					spread.MinCount, spread.MaxCount, spread.MeanCount)
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
