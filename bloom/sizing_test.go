package bloom

import "testing"

func TestSizingMode(t *testing.T) {
	target := TargetConfig(1000, 0.01)
	if target.Mode() != SizingTarget {
		t.Fatalf("TargetConfig.Mode() = %v, want SizingTarget", target.Mode())
	}
	spec, ok := target.Target()
	if !ok {
		t.Fatal("Target() = false for target config")
	}
	if spec.Capacity != 1000 || spec.FPR != 0.01 {
		t.Fatalf("Target() = %+v, want n=1000 p=0.01", spec)
	}
	if _, ok := target.Explicit(); ok {
		t.Fatal("Explicit() = true for target config")
	}

	explicit := ExplicitConfig(256, 5)
	if explicit.Mode() != SizingExplicit {
		t.Fatalf("ExplicitConfig.Mode() = %v, want SizingExplicit", explicit.Mode())
	}
	es, ok := explicit.Explicit()
	if !ok {
		t.Fatal("Explicit() = false for explicit config")
	}
	if es.Bits != 256 || es.HashCount != 5 {
		t.Fatalf("Explicit() = %+v, want m=256 k=5", es)
	}
	if _, ok := explicit.Target(); ok {
		t.Fatal("Target() = true for explicit config")
	}

	incomplete := ExplicitConfig(0, 4)
	if incomplete.Mode() != SizingExplicit {
		t.Fatalf("ExplicitConfig(0, k).Mode() = %v, want SizingExplicit", incomplete.Mode())
	}
	if _, ok := incomplete.Target(); ok {
		t.Fatal("Target() = true for incomplete explicit config")
	}
	ies, ok := incomplete.Explicit()
	if !ok {
		t.Fatal("Explicit() = false for incomplete explicit config")
	}
	if ies.Bits != 0 || ies.HashCount != 4 {
		t.Fatalf("Explicit() = %+v, want m=0 k=4", ies)
	}
	if err := incomplete.Validate(); err != ErrInvalidBits {
		t.Fatalf("Validate() = %v, want ErrInvalidBits", err)
	}
}

func TestSizingBoundsResolved(t *testing.T) {
	defaults := DefaultSizingBounds()
	if defaults.MinBits != defaultMinBits || defaults.MaxHashCount != defaultMaxHashCount {
		t.Fatalf("DefaultSizingBounds() = %+v", defaults)
	}

	empty := SizingBounds{}.Resolved()
	if empty.MinBits != defaultMinBits || empty.MaxHashCount != defaultMaxHashCount {
		t.Fatalf("empty bounds Resolved() = %+v", empty)
	}

	custom := SizingBounds{MinBits: 512, MaxHashCount: 12}.Resolved()
	if custom.MinBits != 512 || custom.MaxHashCount != 12 {
		t.Fatalf("custom bounds Resolved() = %+v", custom)
	}
}

func TestWithSizingBounds(t *testing.T) {
	bounds := SizingBounds{MinBits: 256, MaxHashCount: 8}
	cfg := TargetConfig(100_000, 0.001, WithSizingBounds(bounds))
	if cfg.Bounds() != bounds {
		t.Fatalf("Bounds() = %+v, want %+v", cfg.Bounds(), bounds)
	}
	m, k, err := cfg.Size()
	if err != nil {
		t.Fatal(err)
	}
	if m < 256 {
		t.Errorf("m = %d, want >= 256", m)
	}
	if k > 8 {
		t.Errorf("k = %d, want <= 8", k)
	}

	base := TargetConfig(1000, 0.01)
	updated := base.WithSizingBounds(bounds)
	if base.MinBits != 0 || base.MaxHashCount != 0 {
		t.Fatalf("WithSizingBounds mutated base: minBits=%d maxK=%d", base.MinBits, base.MaxHashCount)
	}
	if updated.Bounds() != bounds {
		t.Fatalf("updated Bounds() = %+v, want %+v", updated.Bounds(), bounds)
	}
}

func TestConfigImmutableUpdaters(t *testing.T) {
	base := TargetConfig(1000, 0.01)
	updated := base.
		WithSeed(99).
		WithMinBits(256).
		WithMaxHashCount(8).
		WithCounterWidth(16).
		WithHashConfig(HashConfig{Strategy: HashMurmur3, Seed: 7})

	if base.Hash.Seed != 0 || base.MinBits != 0 || base.CounterWidth != 0 {
		t.Fatalf("updaters mutated base config: %+v", base)
	}
	if updated.Hash.Strategy != HashMurmur3 || updated.Hash.Seed != 7 {
		t.Fatalf("WithHashConfig = %+v", updated.Hash)
	}
	if updated.MinBits != 256 || updated.MaxHashCount != 8 || updated.CounterWidth != 16 {
		t.Fatalf("bounds/counter = minBits=%d maxK=%d width=%d", updated.MinBits, updated.MaxHashCount, updated.CounterWidth)
	}
}
