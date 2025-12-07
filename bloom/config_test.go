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
}

func TestHashConfigString(t *testing.T) {
	if got := (HashConfig{}).String(); got != "fnv" {
		t.Errorf("default HashConfig.String() = %q, want fnv", got)
	}
	if got := (HashConfig{Strategy: HashMurmur3, Seed: 7}).String(); got != "murmur3 seed=7" {
		t.Errorf("seeded HashConfig.String() = %q", got)
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
