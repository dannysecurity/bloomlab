package bloom

import "testing"

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr error
	}{
		{
			name:    "valid target",
			cfg:     TargetConfig(1000, 0.01),
			wantErr: nil,
		},
		{
			name:    "valid explicit",
			cfg:     ExplicitConfig(128, 4),
			wantErr: nil,
		},
		{
			name:    "explicit zero hash count",
			cfg:     ExplicitConfig(64, 0),
			wantErr: nil,
		},
		{
			name:    "explicit zero bits",
			cfg:     ExplicitConfig(0, 4),
			wantErr: ErrInvalidBits,
		},
		{
			name:    "zero capacity",
			cfg:     TargetConfig(0, 0.01),
			wantErr: ErrInvalidCapacity,
		},
		{
			name:    "zero fpr",
			cfg:     TargetConfig(100, 0),
			wantErr: ErrInvalidFPR,
		},
		{
			name:    "fpr one",
			cfg:     TargetConfig(100, 1.0),
			wantErr: ErrInvalidFPR,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if err != tt.wantErr {
				t.Errorf("Validate() = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfigSize(t *testing.T) {
	tests := []struct {
		name   string
		cfg    Config
		wantM  uint64
		wantK  uint
		wantErr error
	}{
		{
			name:  "explicit sizing",
			cfg:   ExplicitConfig(256, 5),
			wantM: 256,
			wantK: 5,
		},
		{
			name:  "explicit zero k defaults to one",
			cfg:   ExplicitConfig(64, 0),
			wantM: 64,
			wantK: 1,
		},
		{
			name:  "target sizing respects min bits",
			cfg:   TargetConfig(10, 0.5),
			wantM: 64,
		},
		{
			name: "custom min bits",
			cfg: Config{
				ExpectedCapacity:  10,
				FalsePositiveRate: 0.5,
				MinBits:           128,
			},
			wantM: 128,
		},
		{
			name: "custom max hash count",
			cfg: Config{
				ExpectedCapacity:  100_000,
				FalsePositiveRate: 0.001,
				MaxHashCount:      8,
			},
			wantK: 8,
		},
		{
			name:    "invalid target",
			cfg:     TargetConfig(0, 0.01),
			wantErr: ErrInvalidCapacity,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, k, err := tt.cfg.Size()
			if err != tt.wantErr {
				t.Fatalf("Size() error = %v, want %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if tt.wantM != 0 && m != tt.wantM {
				t.Errorf("m = %d, want %d", m, tt.wantM)
			}
			if tt.wantK != 0 && k != tt.wantK {
				t.Errorf("k = %d, want %d", k, tt.wantK)
			}
		})
	}
}

func TestConfigString(t *testing.T) {
	target := TargetConfig(1000, 0.01)
	if s := target.String(); s == "" || s[:6] != "target" {
		t.Errorf("TargetConfig.String() = %q, want target prefix", s)
	}

	explicit := ExplicitConfig(128, 4)
	if s := explicit.String(); s != "explicit m=128 k=4 hash=fnv" {
		t.Errorf("ExplicitConfig.String() = %q", s)
	}

	wide16 := ExplicitConfig(128, 4, WithCounterWidth(16))
	if s := wide16.String(); s != "explicit m=128 k=4 hash=fnv counter-width=16" {
		t.Errorf("wide ExplicitConfig.String() = %q", s)
	}

	wide32 := ExplicitConfig(128, 4, WithCounterWidth(32))
	if s := wide32.String(); s != "explicit m=128 k=4 hash=fnv counter-width=32" {
		t.Errorf("extra-wide ExplicitConfig.String() = %q", s)
	}

	wide64 := ExplicitConfig(128, 4, WithCounterWidth(64))
	if s := wide64.String(); s != "explicit m=128 k=4 hash=fnv counter-width=64" {
		t.Errorf("ultra-wide ExplicitConfig.String() = %q", s)
	}

	invalid := TargetConfig(0, 0.01)
	if s := invalid.String(); s[:7] != "invalid" {
		t.Errorf("invalid config String() = %q", s)
	}
}

func TestNewFilterFromConfig(t *testing.T) {
	cfg := TargetConfig(500, 0.02)
	f, err := NewFilter(cfg)
	if err != nil {
		t.Fatal(err)
	}
	m, k, err := cfg.Size()
	if err != nil {
		t.Fatal(err)
	}
	if f.BitCount() != m || f.HashCount() != k {
		t.Errorf("filter sizing m=%d k=%d, want m=%d k=%d", f.BitCount(), f.HashCount(), m, k)
	}
}

func TestNewCountingFilterFromConfig(t *testing.T) {
	cfg := ExplicitConfig(64, 3)
	cf, err := NewCountingFilter(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if cf.BitCount() != 64 || cf.HashCount() != 3 {
		t.Errorf("BitCount=%d HashCount=%d", cf.BitCount(), cf.HashCount())
	}
}

func TestConfigOptions(t *testing.T) {
	cfg := TargetConfig(1000, 0.01, WithHash(HashMurmur3), WithSeed(42))
	if cfg.Hash.Strategy != HashMurmur3 || cfg.Hash.Seed != 42 {
		t.Fatalf("Hash = %+v, want murmur3 seed=42", cfg.Hash)
	}
	h := cfg.Hasher()
	if h.Strategy() != HashMurmur3 {
		t.Fatalf("Hasher strategy = %v, want murmur3", h.Strategy())
	}

	bounded := TargetConfig(100_000, 0.001, WithMinBits(256), WithMaxHashCount(8))
	if bounded.MinBits != 256 || bounded.MaxHashCount != 8 {
		t.Fatalf("bounds = minBits=%d maxK=%d, want 256 and 8", bounded.MinBits, bounded.MaxHashCount)
	}
	m, k, err := bounded.Size()
	if err != nil {
		t.Fatal(err)
	}
	if m < 256 {
		t.Errorf("m = %d, want >= 256", m)
	}
	if k > 8 {
		t.Errorf("k = %d, want <= 8", k)
	}

	hashCfg := TargetConfig(500, 0.01, WithHashConfig(HashConfig{Strategy: HashMurmur3, Seed: 9}))
	if hashCfg.Hash.Strategy != HashMurmur3 || hashCfg.Hash.Seed != 9 {
		t.Fatalf("WithHashConfig = %+v", hashCfg.Hash)
	}

	wide := ExplicitConfig(64, 2, WithCounterWidth(16))
	if wide.CounterWidth != 16 {
		t.Fatalf("CounterWidth = %d, want 16", wide.CounterWidth)
	}
}

func TestHashConfigString(t *testing.T) {
	if got := (HashConfig{}).String(); got != "fnv" {
		t.Errorf("default HashConfig.String() = %q, want fnv", got)
	}
	if got := (HashConfig{Strategy: HashMurmur3, Seed: 7}).String(); got != "murmur3 seed=7" {
		t.Errorf("seeded HashConfig.String() = %q", got)
	}
}

func TestConfigWithBounds(t *testing.T) {
	cfg := TargetConfig(100_000, 0.001, WithMinBits(256), WithMaxHashCount(8))
	updated := cfg.WithFalsePositiveRate(0.05)
	if updated.FalsePositiveRate != 0.05 || cfg.FalsePositiveRate != 0.001 {
		t.Fatalf("WithFalsePositiveRate mutated original: orig=%g updated=%g", cfg.FalsePositiveRate, updated.FalsePositiveRate)
	}

	resized := cfg.WithExpectedCapacity(50_000)
	if resized.ExpectedCapacity != 50_000 || cfg.ExpectedCapacity != 100_000 {
		t.Fatalf("WithExpectedCapacity mutated original: orig=%d updated=%d", cfg.ExpectedCapacity, resized.ExpectedCapacity)
	}

	hashed := cfg.WithHashStrategy(HashXXHash)
	if hashed.Hash.Strategy != HashXXHash || cfg.Hash.Strategy != HashFNV {
		t.Fatalf("WithHashStrategy mutated original: orig=%v updated=%v", cfg.Hash.Strategy, hashed.Hash.Strategy)
	}
}

func TestConfigMatchesLegacyConstructors(t *testing.T) {
	legacy, err := New(5000, 0.01)
	if err != nil {
		t.Fatal(err)
	}
	fromCfg, err := NewFilter(TargetConfig(5000, 0.01))
	if err != nil {
		t.Fatal(err)
	}
	if legacy.BitCount() != fromCfg.BitCount() || legacy.HashCount() != fromCfg.HashCount() {
		t.Errorf("legacy vs config: m=%d/%d k=%d/%d",
			legacy.BitCount(), fromCfg.BitCount(),
			legacy.HashCount(), fromCfg.HashCount())
	}

	legacyCF, err := NewCounting(128, 4)
	if err != nil {
		t.Fatal(err)
	}
	fromCfgCF, err := NewCountingFilter(ExplicitConfig(128, 4))
	if err != nil {
		t.Fatal(err)
	}
	if legacyCF.BitCount() != fromCfgCF.BitCount() || legacyCF.HashCount() != fromCfgCF.HashCount() {
		t.Errorf("counting legacy vs config mismatch")
	}
}

func TestConfigWithRecommendedHash(t *testing.T) {
	cfg := TargetConfig(2000, 0.01)
	tuned, err := cfg.WithRecommendedHash(RecommendedHashOptions{
		Samples: 1000,
		Strategies: []Strategy{HashMurmur3, HashXXHash, HashWyhash},
		Seeds:   []uint64{0, 7},
	})
	if err != nil {
		t.Fatal(err)
	}
	if tuned.Hash.Strategy == 0 && tuned.Hash.Seed == 0 {
		t.Fatal("expected non-default recommended hash")
	}
	f, err := NewFilter(tuned)
	if err != nil {
		t.Fatal(err)
	}
	f.Add([]byte("recommended"))
	if !f.Contains([]byte("recommended")) {
		t.Fatal("recommended-hash filter should contain inserted key")
	}
}
