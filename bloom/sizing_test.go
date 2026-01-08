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
