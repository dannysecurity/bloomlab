package setup_test

import (
	"testing"

	"github.com/dannysecurity/bloomlab/bloom"
	"github.com/dannysecurity/bloomlab/bloom/setup"
)

func TestTargetFilterConfig(t *testing.T) {
	fc, err := setup.Target(10_000, 0.01,
		setup.WithHash(bloom.HashMurmur3),
		setup.WithSeed(42),
	).FilterConfig()
	if err != nil {
		t.Fatal(err)
	}
	if fc.Mode() != bloom.SizingTarget {
		t.Fatalf("Mode() = %v, want target", fc.Mode())
	}
	spec, ok := fc.Target()
	if !ok || spec.Capacity != 10_000 || spec.FPR != 0.01 {
		t.Fatalf("Target() = %+v, ok=%v", spec, ok)
	}
	if fc.Hash.Strategy != bloom.HashMurmur3 || fc.Hash.Seed != 42 {
		t.Fatalf("Hash = %+v, want murmur3 seed=42", fc.Hash)
	}
}

func TestExplicitFilterConfig(t *testing.T) {
	fc, err := setup.Explicit(1024, 4).FilterConfig()
	if err != nil {
		t.Fatal(err)
	}
	if fc.Mode() != bloom.SizingExplicit {
		t.Fatalf("Mode() = %v, want explicit", fc.Mode())
	}
	m, k, err := fc.Size()
	if err != nil {
		t.Fatal(err)
	}
	if m != 1024 || k != 4 {
		t.Fatalf("Size() = (%d, %d), want (1024, 4)", m, k)
	}
}

func TestCountingConfig(t *testing.T) {
	cc, err := setup.Explicit(512, 3, setup.WithCounterWidth(16)).CountingConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cc.CounterWidth != 16 {
		t.Fatalf("CounterWidth = %d, want 16", cc.CounterWidth)
	}
	if err := cc.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestCountingConfig2Bit(t *testing.T) {
	cc, err := setup.Explicit(512, 3, setup.WithCounterWidth(2)).CountingConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cc.CounterWidth != 2 {
		t.Fatalf("CounterWidth = %d, want 2", cc.CounterWidth)
	}
}

func TestCountingConfig4Bit(t *testing.T) {
	cc, err := setup.Explicit(512, 3, setup.WithCounterWidth(4)).CountingConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cc.CounterWidth != 4 {
		t.Fatalf("CounterWidth = %d, want 4", cc.CounterWidth)
	}
}

func TestInvalidCounterWidth(t *testing.T) {
	_, err := setup.Target(100, 0.01, setup.WithCounterWidth(12)).CountingConfig()
	if err != bloom.ErrInvalidCounterWidth {
		t.Fatalf("CountingConfig() error = %v, want ErrInvalidCounterWidth", err)
	}
}

func TestFilterRoundTrip(t *testing.T) {
	f, err := setup.Target(1000, 0.01).Filter()
	if err != nil {
		t.Fatal(err)
	}
	key := []byte("setup-test")
	f.Add(key)
	if !f.Contains(key) {
		t.Fatal("expected key to be present after Add")
	}
}

func TestCountingFilterRoundTrip(t *testing.T) {
	cf, err := setup.Target(1000, 0.01).CountingFilter()
	if err != nil {
		t.Fatal(err)
	}
	key := []byte("counting-setup")
	if err := cf.Add(key); err != nil {
		t.Fatal(err)
	}
	cf.Remove(key)
	if cf.Contains(key) {
		t.Fatal("expected key absent after Remove")
	}
}

func TestPlanTarget(t *testing.T) {
	plan, err := setup.Target(10_000, 0.01).Plan()
	if err != nil {
		t.Fatal(err)
	}
	if plan.ExpectedCapacity != 10_000 || plan.TargetFPR != 0.01 {
		t.Fatalf("plan = %+v", plan)
	}
	if plan.Bits == 0 || plan.HashCount == 0 {
		t.Fatalf("expected positive m and k, got m=%d k=%d", plan.Bits, plan.HashCount)
	}
}

func TestPlanExplicitRejects(t *testing.T) {
	_, err := setup.Explicit(256, 4).Plan()
	if err != bloom.ErrInvalidCapacity {
		t.Fatalf("Plan() error = %v, want ErrInvalidCapacity", err)
	}
}

func TestSizingBounds(t *testing.T) {
	fc, err := setup.Target(100_000, 0.001,
		setup.WithSizingBounds(bloom.SizingBounds{MinBits: 256, MaxHashCount: 8}),
	).FilterConfig()
	if err != nil {
		t.Fatal(err)
	}
	m, k, err := fc.Size()
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

func TestFromSizing(t *testing.T) {
	sizing := bloom.TargetSizing(5000, 0.05, bloom.SizingBounds{})
	fc, err := setup.FromSizing(sizing, setup.WithHash(bloom.HashXXHash)).FilterConfig()
	if err != nil {
		t.Fatal(err)
	}
	if fc.Hash.Strategy != bloom.HashXXHash {
		t.Fatalf("Hash.Strategy = %v, want xxhash", fc.Hash.Strategy)
	}
}

func TestApplyChain(t *testing.T) {
	base := setup.Target(1000, 0.01)
	updated := base.Apply(setup.WithSeed(99), setup.WithHash(bloom.HashWyhash))
	fc, err := updated.FilterConfig()
	if err != nil {
		t.Fatal(err)
	}
	if fc.Hash.Seed != 99 || fc.Hash.Strategy != bloom.HashWyhash {
		t.Fatalf("Hash = %+v, want wyhash seed=99", fc.Hash)
	}
	orig, err := base.FilterConfig()
	if err != nil {
		t.Fatal(err)
	}
	if orig.Hash.Seed != 0 {
		t.Fatalf("Apply mutated base builder: seed = %d", orig.Hash.Seed)
	}
}

func TestValidationErrors(t *testing.T) {
	tests := []struct {
		name string
		b    *setup.Builder
		want error
	}{
		{
			name: "zero capacity",
			b:    setup.Target(0, 0.01),
			want: bloom.ErrInvalidCapacity,
		},
		{
			name: "invalid fpr",
			b:    setup.Target(100, 1.5),
			want: bloom.ErrInvalidFPR,
		},
		{
			name: "zero bits",
			b:    setup.Explicit(0, 4),
			want: bloom.ErrInvalidBits,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.b.FilterConfig()
			if err != tt.want {
				t.Fatalf("FilterConfig() error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestMatchesFilterConfigHelpers(t *testing.T) {
	legacy, err := bloom.BuildFilterFromSizing(
		bloom.TargetSizing(8000, 0.02, bloom.SizingBounds{MinBits: 128, MaxHashCount: 16}),
		bloom.WithFilterHash(bloom.HashHighway),
		bloom.WithFilterSeed(7),
	)
	if err != nil {
		t.Fatal(err)
	}
	built, err := setup.Target(8000, 0.02,
		setup.WithSizingBounds(bloom.SizingBounds{MinBits: 128, MaxHashCount: 16}),
		setup.WithHash(bloom.HashHighway),
		setup.WithSeed(7),
	).FilterConfig()
	if err != nil {
		t.Fatal(err)
	}
	lm, lk, _ := legacy.Size()
	bm, bk, _ := built.Size()
	if lm != bm || lk != bk {
		t.Fatalf("Size mismatch: legacy (%d,%d) vs setup (%d,%d)", lm, lk, bm, bk)
	}
	if legacy.Hash != built.Hash {
		t.Fatalf("Hash mismatch: legacy %+v vs setup %+v", legacy.Hash, built.Hash)
	}
}
