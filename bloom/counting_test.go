package bloom

import (
	"testing"
	"testing/quick"
)

func TestCountingFilterCounterBytes(t *testing.T) {
	cf, err := NewCounting(128, 4)
	if err != nil {
		t.Fatal(err)
	}
	if got := cf.CounterBytes(); got != 128 {
		t.Fatalf("CounterBytes() = %d, want 128", got)
	}
}

func TestCountingFilterAddRemove(t *testing.T) {
	cf, err := NewCountingFromTarget(256, 0.01)
	if err != nil {
		t.Fatal(err)
	}

	key := []byte("deletable")
	if err := cf.Add(key); err != nil {
		t.Fatal(err)
	}
	if !cf.Contains(key) {
		t.Fatal("expected key after add")
	}

	if !cf.Remove(key) {
		t.Fatal("remove should succeed for present key")
	}
	if cf.Contains(key) {
		t.Error("key should be absent after remove")
	}
}

func TestCountingFilterRemoveAbsent(t *testing.T) {
	cf, err := NewCounting(128, 4)
	if err != nil {
		t.Fatal(err)
	}
	if cf.Remove([]byte("never-added")) {
		t.Error("remove of absent key should return false")
	}
}

func TestCountingFilterOverflow(t *testing.T) {
	cf, err := NewCounting(8, 1)
	if err != nil {
		t.Fatal(err)
	}

	key := []byte("x")
	for i := 0; i < 255; i++ {
		if err := cf.Add(key); err != nil {
			t.Fatalf("add %d: %v", i, err)
		}
	}
	if err := cf.Add(key); err != ErrCounterOverflow {
		t.Errorf("expected ErrCounterOverflow, got %v", err)
	}
}

func TestCountingFilterCounterHeadroom(t *testing.T) {
	cf, err := NewCounting(64, 2)
	if err != nil {
		t.Fatal(err)
	}
	if got := cf.CounterHeadroom(); got != 255 {
		t.Fatalf("empty filter CounterHeadroom() = %d, want 255", got)
	}

	key := []byte("dup")
	for range 3 {
		if err := cf.Add(key); err != nil {
			t.Fatal(err)
		}
	}
	if got := cf.CounterHeadroom(); got != 252 {
		t.Fatalf("after three adds CounterHeadroom() = %d, want 252", got)
	}

	for i := 0; i < 252; i++ {
		if err := cf.Add(key); err != nil {
			t.Fatalf("add %d: %v", i, err)
		}
	}
	if got := cf.CounterHeadroom(); got != 0 {
		t.Fatalf("at saturation CounterHeadroom() = %d, want 0", got)
	}
	if err := cf.Add(key); err != ErrCounterOverflow {
		t.Errorf("expected ErrCounterOverflow, got %v", err)
	}
}

func TestCountingFilterCounterLimit(t *testing.T) {
	narrow, err := NewCounting(64, 2)
	if err != nil {
		t.Fatal(err)
	}
	if got := narrow.CounterLimit(); got != 255 {
		t.Fatalf("8-bit CounterLimit() = %d, want 255", got)
	}

	wide, err := NewCountingFilter(ExplicitConfig(64, 2, WithCounterWidth(16)))
	if err != nil {
		t.Fatal(err)
	}
	if got := wide.CounterLimit(); got != 65535 {
		t.Fatalf("16-bit CounterLimit() = %d, want 65535", got)
	}
}

func TestCountingFilterWideCounterWidth(t *testing.T) {
	cf, err := NewCountingFilter(ExplicitConfig(8, 1, WithCounterWidth(16)))
	if err != nil {
		t.Fatal(err)
	}
	if cf.CounterWidth() != 16 {
		t.Fatalf("CounterWidth() = %d, want 16", cf.CounterWidth())
	}
	if got := cf.CounterBytes(); got != 16 {
		t.Fatalf("CounterBytes() = %d, want 16", got)
	}

	key := []byte("x")
	for i := 0; i < 256; i++ {
		if err := cf.Add(key); err != nil {
			t.Fatalf("add %d: %v", i, err)
		}
	}
	if got := cf.MaxCounter(); got != 256 {
		t.Fatalf("MaxCounter() = %d, want 256", got)
	}
	if !cf.Contains(key) {
		t.Fatal("expected key after 256 wide adds")
	}
	if !cf.Remove(key) {
		t.Fatal("remove should succeed")
	}
	if !cf.Contains(key) {
		t.Fatal("key should still appear present after one of 256 adds removed")
	}
}

func TestCountingFilterWideCounterOverflow(t *testing.T) {
	cf, err := NewCountingFilter(ExplicitConfig(8, 1, WithCounterWidth(16)))
	if err != nil {
		t.Fatal(err)
	}

	key := []byte("x")
	for i := 0; i < 65535; i++ {
		if err := cf.Add(key); err != nil {
			t.Fatalf("add %d: %v", i, err)
		}
	}
	if got := cf.MaxCounter(); got != 65535 {
		t.Fatalf("MaxCounter() = %d, want 65535", got)
	}
	if err := cf.Add(key); err != ErrCounterOverflow {
		t.Errorf("expected ErrCounterOverflow, got %v", err)
	}
}

func TestCountingFilterExtraWideCounterWidth(t *testing.T) {
	cf, err := NewCountingFilter(ExplicitConfig(8, 1, WithCounterWidth(32)))
	if err != nil {
		t.Fatal(err)
	}
	if cf.CounterWidth() != 32 {
		t.Fatalf("CounterWidth() = %d, want 32", cf.CounterWidth())
	}
	if got := cf.CounterBytes(); got != 32 {
		t.Fatalf("CounterBytes() = %d, want 32", got)
	}
	if got := cf.CounterLimit(); got != 4294967295 {
		t.Fatalf("CounterLimit() = %d, want 4294967295", got)
	}

	key := []byte("x")
	for i := 0; i < 512; i++ {
		if err := cf.Add(key); err != nil {
			t.Fatalf("add %d: %v", i, err)
		}
	}
	if got := cf.MaxCounter(); got != 512 {
		t.Fatalf("MaxCounter() = %d, want 512", got)
	}
	if !cf.Contains(key) {
		t.Fatal("expected key after 512 wide adds")
	}
	if !cf.Remove(key) {
		t.Fatal("remove should succeed")
	}
	if !cf.Contains(key) {
		t.Fatal("key should still appear present after one of 512 adds removed")
	}
}

func TestCountingFilterUltraWideCounterWidth(t *testing.T) {
	cf, err := NewCountingFilter(ExplicitConfig(8, 1, WithCounterWidth(64)))
	if err != nil {
		t.Fatal(err)
	}
	if cf.CounterWidth() != 64 {
		t.Fatalf("CounterWidth() = %d, want 64", cf.CounterWidth())
	}
	if got := cf.CounterBytes(); got != 64 {
		t.Fatalf("CounterBytes() = %d, want 64", got)
	}
	if got := cf.CounterLimit(); got != 18446744073709551615 {
		t.Fatalf("CounterLimit() = %d, want 18446744073709551615", got)
	}

	key := []byte("x")
	for i := 0; i < 1024; i++ {
		if err := cf.Add(key); err != nil {
			t.Fatalf("add %d: %v", i, err)
		}
	}
	if got := cf.MaxCounter(); got != 1024 {
		t.Fatalf("MaxCounter() = %d, want 1024", got)
	}
	if !cf.Contains(key) {
		t.Fatal("expected key after 1024 ultra-wide adds")
	}
	if !cf.Remove(key) {
		t.Fatal("remove should succeed")
	}
	if !cf.Contains(key) {
		t.Fatal("key should still appear present after one of 1024 adds removed")
	}
}

func TestCountingFilterInvalidCounterWidth(t *testing.T) {
	_, err := NewCountingFilter(ExplicitConfig(64, 2, WithCounterWidth(24)))
	if err != ErrInvalidCounterWidth {
		t.Errorf("error = %v, want %v", err, ErrInvalidCounterWidth)
	}
}

func TestCountingFilterClear(t *testing.T) {
	cf, err := NewCountingFromTarget(64, 0.05)
	if err != nil {
		t.Fatal(err)
	}

	keys := [][]byte{[]byte("a"), []byte("b")}
	for _, key := range keys {
		if err := cf.Add(key); err != nil {
			t.Fatal(err)
		}
	}
	if cf.ApproximateCount() != 2 {
		t.Fatalf("before clear count = %d, want 2", cf.ApproximateCount())
	}

	cf.Clear()

	if cf.ApproximateCount() != 0 {
		t.Errorf("after clear count = %d, want 0", cf.ApproximateCount())
	}
	if cf.FillRatio() != 0 {
		t.Errorf("after clear fill ratio = %v, want 0", cf.FillRatio())
	}
	for _, key := range keys {
		if cf.Contains(key) {
			t.Errorf("Contains(%q) = true after clear, want false", key)
		}
	}
	if cf.TheoryFPR() != 0 {
		t.Errorf("TheoryFPR() = %v after clear, want 0", cf.TheoryFPR())
	}

	// Filter remains usable after clear.
	if err := cf.Add(keys[0]); err != nil {
		t.Fatal(err)
	}
	if !cf.Contains(keys[0]) {
		t.Error("expected key present after re-add post-clear")
	}
}

func TestCountingFilterMaxCounter(t *testing.T) {
	cf, err := NewCounting(64, 2)
	if err != nil {
		t.Fatal(err)
	}
	if got := cf.MaxCounter(); got != 0 {
		t.Fatalf("empty filter MaxCounter() = %d, want 0", got)
	}

	key := []byte("dup")
	for range 3 {
		if err := cf.Add(key); err != nil {
			t.Fatal(err)
		}
	}
	if got := cf.MaxCounter(); got != 3 {
		t.Fatalf("after three adds MaxCounter() = %d, want 3", got)
	}

	cf.Clear()
	if got := cf.MaxCounter(); got != 0 {
		t.Fatalf("after clear MaxCounter() = %d, want 0", got)
	}
}

func TestCountingFilterOccupiedCount(t *testing.T) {
	cf, err := NewCounting(64, 2)
	if err != nil {
		t.Fatal(err)
	}
	if got := cf.OccupiedCount(); got != 0 {
		t.Fatalf("empty filter OccupiedCount() = %d, want 0", got)
	}

	if err := cf.Add([]byte("a")); err != nil {
		t.Fatal(err)
	}
	occupied := cf.OccupiedCount()
	if occupied == 0 || occupied > cf.BitCount() {
		t.Fatalf("after add OccupiedCount() = %d, want in (0, %d]", occupied, cf.BitCount())
	}

	if err := cf.Add([]byte("b")); err != nil {
		t.Fatal(err)
	}
	if got := cf.OccupiedCount(); got < occupied {
		t.Fatalf("after second add OccupiedCount() = %d, want >= %d", got, occupied)
	}

	if !cf.Remove([]byte("a")) {
		t.Fatal("remove should succeed")
	}
	if got := cf.OccupiedCount(); got == 0 {
		t.Fatal("after removing one of two keys OccupiedCount() = 0, want > 0")
	}

	cf.Clear()
	if got := cf.OccupiedCount(); got != 0 {
		t.Fatalf("after clear OccupiedCount() = %d, want 0", got)
	}
}

func TestCountingFilterFillRatio(t *testing.T) {
	cf, err := NewCounting(64, 2)
	if err != nil {
		t.Fatal(err)
	}
	if ratio := cf.FillRatio(); ratio != 0 {
		t.Fatalf("empty filter fill ratio = %v, want 0", ratio)
	}

	if err := cf.Add([]byte("a")); err != nil {
		t.Fatal(err)
	}
	ratio := cf.FillRatio()
	if ratio <= 0 || ratio > 1 {
		t.Fatalf("after add fill ratio = %v, want in (0, 1]", ratio)
	}
	if want := float64(cf.OccupiedCount()) / float64(cf.BitCount()); ratio != want {
		t.Fatalf("FillRatio() = %v, want OccupiedCount/BitCount = %v", ratio, want)
	}

	if !cf.Remove([]byte("a")) {
		t.Fatal("remove should succeed")
	}
	if ratio := cf.FillRatio(); ratio != 0 {
		t.Fatalf("after remove fill ratio = %v, want 0", ratio)
	}
}

func TestCountingFilterInvalidParams(t *testing.T) {
	tests := []struct {
		name    string
		newFn   func() error
		wantErr error
	}{
		{
			name: "NewCounting zero bits",
			newFn: func() error {
				_, err := NewCounting(0, 4)
				return err
			},
			wantErr: ErrInvalidBits,
		},
		{
			name: "NewCountingFromTarget zero capacity",
			newFn: func() error {
				_, err := NewCountingFromTarget(0, 0.01)
				return err
			},
			wantErr: ErrInvalidCapacity,
		},
		{
			name: "NewCountingFromTarget zero fpr",
			newFn: func() error {
				_, err := NewCountingFromTarget(100, 0)
				return err
			},
			wantErr: ErrInvalidFPR,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.newFn(); err != tt.wantErr {
				t.Errorf("error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestCountingFilterDuplicateAdds(t *testing.T) {
	cf, err := NewCountingFromTarget(64, 0.05)
	if err != nil {
		t.Fatal(err)
	}

	key := []byte("dup")
	for range 3 {
		if err := cf.Add(key); err != nil {
			t.Fatal(err)
		}
	}
	if !cf.Remove(key) {
		t.Fatal("first remove should succeed")
	}
	if !cf.Contains(key) {
		t.Error("key should still appear present after one of three adds removed")
	}
	if got := cf.ApproximateCount(); got != 2 {
		t.Errorf("ApproximateCount() = %d, want 2 after one remove of three adds", got)
	}
}

func TestCountingFilterApproximateCount(t *testing.T) {
	cf, err := NewCountingFromTarget(64, 0.05)
	if err != nil {
		t.Fatal(err)
	}
	if cf.ApproximateCount() != 0 {
		t.Fatalf("empty filter count = %d, want 0", cf.ApproximateCount())
	}

	key := []byte("tracked")
	for range 2 {
		if err := cf.Add(key); err != nil {
			t.Fatal(err)
		}
	}
	if got := cf.ApproximateCount(); got != 2 {
		t.Fatalf("after two adds count = %d, want 2", got)
	}
	if !cf.Remove(key) {
		t.Fatal("remove should succeed")
	}
	if got := cf.ApproximateCount(); got != 1 {
		t.Fatalf("after one remove count = %d, want 1", got)
	}
}

func TestCountingFilterTheoryFPR(t *testing.T) {
	cf, err := NewCountingFromTarget(1000, 0.01)
	if err != nil {
		t.Fatal(err)
	}
	if cf.TheoryFPR() != 0 {
		t.Fatalf("empty filter FPR = %v, want 0", cf.TheoryFPR())
	}

	for i := 0; i < 1000; i++ {
		if err := cf.Add([]byte{byte(i >> 8), byte(i)}); err != nil {
			t.Fatal(err)
		}
	}
	fpr := cf.TheoryFPR()
	if fpr <= 0 || fpr > 0.05 {
		t.Fatalf("TheoryFPR() = %v, want in (0, 0.05] for n=1000 p=0.01", fpr)
	}
}

func TestCountingFilterAddContainsProperty(t *testing.T) {
	prop := func(capacity uint16, key []byte) bool {
		if capacity == 0 {
			capacity = 1
		}
		if len(key) == 0 {
			return true
		}
		cf, err := NewCountingFromTarget(uint64(capacity), 0.05)
		if err != nil {
			return false
		}
		if err := cf.Add(key); err != nil {
			return false
		}
		return cf.Contains(key)
	}
	cfg := &quick.Config{MaxCount: 200}
	if err := quick.Check(prop, cfg); err != nil {
		t.Error(err)
	}
}

func TestCountingFilterAddRemoveProperty(t *testing.T) {
	prop := func(capacity uint16, key []byte) bool {
		if capacity == 0 {
			capacity = 1
		}
		if len(key) == 0 {
			return true
		}
		cf, err := NewCountingFromTarget(uint64(capacity), 0.05)
		if err != nil {
			return false
		}
		if err := cf.Add(key); err != nil {
			return false
		}
		if !cf.Remove(key) {
			return false
		}
		return !cf.Contains(key)
	}
	cfg := &quick.Config{MaxCount: 200}
	if err := quick.Check(prop, cfg); err != nil {
		t.Error(err)
	}
}

func TestCountingFilterFalsePositiveRate(t *testing.T) {
	const n = 5000
	const trials = 5000
	f, err := newCountingEmpiricalFilter(TargetConfig(n, 0.01))
	if err != nil {
		t.Fatal(err)
	}
	assertEmpiricalFPR(t, f, n, trials, 0.05)
}
