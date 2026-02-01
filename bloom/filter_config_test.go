package bloom

import "testing"

func TestSizingConfigSize(t *testing.T) {
	tests := []struct {
		name   string
		sizing SizingConfig
		wantM  uint64
		wantK  uint
		wantErr error
	}{
		{
			name:  "explicit",
			sizing: ExplicitSizing(256, 5),
			wantM: 256,
			wantK: 5,
		},
		{
			name:  "explicit zero k defaults to one",
			sizing: ExplicitSizing(64, 0),
			wantM: 64,
			wantK: 1,
		},
		{
			name:  "target respects min bits",
			sizing: TargetSizing(10, 0.5, SizingBounds{}),
			wantM: 64,
		},
		{
			name: "target custom bounds",
			sizing: TargetSizing(100_000, 0.001, SizingBounds{MinBits: 256, MaxHashCount: 8}),
		},
		{
			name:    "invalid target capacity",
			sizing:  TargetSizing(0, 0.01, SizingBounds{}),
			wantErr: ErrInvalidCapacity,
		},
		{
			name:    "invalid explicit bits",
			sizing:  ExplicitSizing(0, 4),
			wantErr: ErrInvalidBits,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, k, err := tt.sizing.Size()
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
			if tt.name == "target custom bounds" {
				if m < 256 {
					t.Errorf("m = %d, want >= 256", m)
				}
				if k > 8 {
					t.Errorf("k = %d, want <= 8", k)
				}
			}
		})
	}
}

func TestBuildFilterConfig(t *testing.T) {
	fc, err := BuildFilterConfig(SizingTarget, TargetSpec{Capacity: 1000, FPR: 0.01}, ExplicitSpec{})
	if err != nil {
		t.Fatal(err)
	}
	if fc.Mode() != SizingTarget {
		t.Fatalf("Mode() = %v, want target", fc.Mode())
	}

	fc, err = BuildFilterConfig(SizingExplicit, TargetSpec{}, ExplicitSpec{Bits: 256, HashCount: 4})
	if err != nil {
		t.Fatal(err)
	}
	if fc.Mode() != SizingExplicit {
		t.Fatalf("Mode() = %v, want explicit", fc.Mode())
	}

	if _, err := BuildFilterConfig(SizingTarget, TargetSpec{Capacity: 0, FPR: 0.01}, ExplicitSpec{}); err != ErrInvalidCapacity {
		t.Fatalf("BuildFilterConfig zero capacity = %v, want ErrInvalidCapacity", err)
	}
	if _, err := BuildFilterConfig(SizingExplicit, TargetSpec{}, ExplicitSpec{Bits: 0, HashCount: 4}); err != ErrInvalidBits {
		t.Fatalf("BuildFilterConfig incomplete explicit = %v, want ErrInvalidBits", err)
	}
}

func TestFilterConfigOptions(t *testing.T) {
	fc := TargetFilter(1000, 0.01, WithFilterHash(HashMurmur3), WithFilterSeed(42))
	if fc.Hash.Strategy != HashMurmur3 || fc.Hash.Seed != 42 {
		t.Fatalf("Hash = %+v, want murmur3 seed=42", fc.Hash)
	}

	bounded := TargetFilter(100_000, 0.001, WithFilterSizingBounds(SizingBounds{MinBits: 256, MaxHashCount: 8}))
	if bounded.Sizing.Target.Bounds.MinBits != 256 || bounded.Sizing.Target.Bounds.MaxHashCount != 8 {
		t.Fatalf("bounds = %+v", bounded.Sizing.Target.Bounds)
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
}

func TestFilterConfigConfigRoundTrip(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
	}{
		{"target", TargetConfig(5000, 0.02, WithHash(HashMurmur3), WithSeed(7))},
		{"explicit", ExplicitConfig(512, 6, WithMinBits(128))},
		{"bounded target", TargetConfig(100_000, 0.001, WithSizingBounds(SizingBounds{MinBits: 256, MaxHashCount: 10}))},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fc := tt.cfg.FilterConfig()
			back := fc.Config()

			if back.Mode() != tt.cfg.Mode() {
				t.Fatalf("Mode() = %v, want %v", back.Mode(), tt.cfg.Mode())
			}
			m1, k1, err := tt.cfg.Size()
			if err != nil {
				t.Fatal(err)
			}
			m2, k2, err := back.Size()
			if err != nil {
				t.Fatal(err)
			}
			if m1 != m2 || k1 != k2 {
				t.Fatalf("Size mismatch: (%d,%d) vs (%d,%d)", m1, k1, m2, k2)
			}
			if back.Hash != tt.cfg.Hash {
				t.Fatalf("Hash = %+v, want %+v", back.Hash, tt.cfg.Hash)
			}
		})
	}
}

func TestFilterConfigFromLegacyConstructors(t *testing.T) {
	legacy, err := New(5000, 0.01)
	if err != nil {
		t.Fatal(err)
	}
	fromFC, err := NewFilterFrom(TargetFilter(5000, 0.01))
	if err != nil {
		t.Fatal(err)
	}
	if legacy.BitCount() != fromFC.BitCount() || legacy.HashCount() != fromFC.HashCount() {
		t.Errorf("legacy vs FilterConfig: m=%d/%d k=%d/%d",
			legacy.BitCount(), fromFC.BitCount(),
			legacy.HashCount(), fromFC.HashCount())
	}
}

func TestCountingConfigRoundTrip(t *testing.T) {
	cfg := ExplicitConfig(128, 4, WithCounterWidth(16))
	cc := cfg.CountingConfig()
	if cc.CounterWidth != 16 {
		t.Fatalf("CounterWidth = %d, want 16", cc.CounterWidth)
	}
	back := cc.Config()
	if back.CounterWidth != 16 || back.Bits != 128 || back.HashCount != 4 {
		t.Fatalf("round trip = %+v", back)
	}

	cf, err := NewCountingFilterFrom(ExplicitCounting(64, 3, WithCountingCounterWidth(32)))
	if err != nil {
		t.Fatal(err)
	}
	if cf.CounterWidth() != 32 || cf.BitCount() != 64 || cf.HashCount() != 3 {
		t.Fatalf("filter sizing width=%d m=%d k=%d", cf.CounterWidth(), cf.BitCount(), cf.HashCount())
	}
}

func TestBuildCountingConfig(t *testing.T) {
	cc, err := BuildCountingConfig(
		SizingExplicit,
		TargetSpec{},
		ExplicitSpec{Bits: 128, HashCount: 4},
		WithCountingCounterWidth(16),
	)
	if err != nil {
		t.Fatal(err)
	}
	if cc.CounterWidth != 16 {
		t.Fatalf("CounterWidth = %d, want 16", cc.CounterWidth)
	}

	if _, err := BuildCountingConfig(
		SizingExplicit,
		TargetSpec{},
		ExplicitSpec{Bits: 128, HashCount: 4},
		WithCountingCounterWidth(48),
	); err != ErrInvalidCounterWidth {
		t.Fatalf("invalid width = %v, want ErrInvalidCounterWidth", err)
	}
}

func TestFilterConfigString(t *testing.T) {
	target := TargetFilter(1000, 0.01)
	if s := target.String(); s == "" || s[:6] != "target" {
		t.Errorf("TargetFilter.String() = %q, want target prefix", s)
	}

	explicit := ExplicitFilter(128, 4)
	if s := explicit.String(); s != "explicit m=128 k=4 hash=fnv" {
		t.Errorf("ExplicitFilter.String() = %q", s)
	}
}

func TestFilterConfigImmutableApply(t *testing.T) {
	base := TargetFilter(1000, 0.01)
	updated := base.Apply(WithFilterHash(HashWyhash), WithFilterSeed(17))
	if base.Hash.Strategy != HashFNV || base.Hash.Seed != 0 {
		t.Fatalf("Apply mutated base: %+v", base.Hash)
	}
	if updated.Hash.Strategy != HashWyhash || updated.Hash.Seed != 17 {
		t.Fatalf("Apply result = %+v", updated.Hash)
	}
}

func TestFilterConfigTargetAccessors(t *testing.T) {
	fc := TargetFilter(5000, 0.01, WithFilterHash(HashMurmur3))
	spec, ok := fc.Target()
	if !ok || spec.Capacity != 5000 || spec.FPR != 0.01 {
		t.Fatalf("Target() = %+v, ok=%v", spec, ok)
	}
	if _, ok := fc.Explicit(); ok {
		t.Fatal("Explicit() should be false for target sizing")
	}
	if fc.ExpectedCapacity() != 5000 || fc.FalsePositiveRate() != 0.01 {
		t.Fatalf("ExpectedCapacity()=%d FalsePositiveRate()=%g", fc.ExpectedCapacity(), fc.FalsePositiveRate())
	}

	explicit := ExplicitFilter(256, 4)
	if explicit.ExpectedCapacity() != 256 || explicit.FalsePositiveRate() != 0 {
		t.Fatalf("explicit accessors = n=%d p=%g", explicit.ExpectedCapacity(), explicit.FalsePositiveRate())
	}

	updated := fc.WithFalsePositiveRate(0.001).WithExpectedCapacity(8000).WithHashStrategy(HashXXHash)
	if updated.FalsePositiveRate() != 0.001 || updated.ExpectedCapacity() != 8000 || updated.Hash.Strategy != HashXXHash {
		t.Fatalf("copy helpers = %+v hash=%v", updated.Sizing.Target, updated.Hash.Strategy)
	}
	if fc.FalsePositiveRate() != 0.01 || fc.Hash.Strategy != HashMurmur3 {
		t.Fatal("copy helpers mutated original config")
	}
}
