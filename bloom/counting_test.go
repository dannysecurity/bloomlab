package bloom

import "testing"

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
			name: "NewCounting zero capacity",
			newFn: func() error {
				_, err := NewCounting(0, 4)
				return err
			},
			wantErr: ErrInvalidCapacity,
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
}
