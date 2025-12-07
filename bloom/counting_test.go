package bloom

import (
	"testing"
	"testing/quick"
)

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
	if ratio := cf.FillRatio(); ratio <= 0 || ratio > 1 {
		t.Fatalf("after add fill ratio = %v, want in (0, 1]", ratio)
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
