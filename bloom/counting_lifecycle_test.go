package bloom

import (
	"testing"
	"testing/quick"
)

type lifecycleOpKind int

const (
	lifecycleAdd lifecycleOpKind = iota
	lifecycleRemove
)

type lifecycleStep struct {
	op  lifecycleOpKind
	key string
}

func applyLifecycle(t *testing.T, cf *CountingFilter, steps []lifecycleStep) {
	t.Helper()
	for i, step := range steps {
		key := []byte(step.key)
		switch step.op {
		case lifecycleAdd:
			if err := cf.Add(key); err != nil {
				t.Fatalf("step %d add %q: %v", i, step.key, err)
			}
		case lifecycleRemove:
			cf.Remove(key)
		default:
			t.Fatalf("step %d: unknown op %d", i, step.op)
		}
	}
}

func TestCountingFilterLifecycle(t *testing.T) {
	tests := []struct {
		name         string
		steps        []lifecycleStep
		wantContains map[string]bool
		wantCount    int
		expectEmpty  bool
	}{
		{
			name: "single add remove",
			steps: []lifecycleStep{
				{lifecycleAdd, "a"},
				{lifecycleRemove, "a"},
			},
			wantContains: map[string]bool{"a": false},
			wantCount:    0,
			expectEmpty:  true,
		},
		{
			name: "duplicate adds partial remove",
			steps: []lifecycleStep{
				{lifecycleAdd, "x"},
				{lifecycleAdd, "x"},
				{lifecycleAdd, "x"},
				{lifecycleRemove, "x"},
			},
			wantContains: map[string]bool{"x": true},
			wantCount:    2,
		},
		{
			name: "interleaved keys",
			steps: []lifecycleStep{
				{lifecycleAdd, "a"},
				{lifecycleAdd, "b"},
				{lifecycleRemove, "a"},
				{lifecycleAdd, "c"},
			},
			wantContains: map[string]bool{"a": false, "b": true, "c": true},
			wantCount:    2,
		},
		{
			name: "remove absent is no-op",
			steps: []lifecycleStep{
				{lifecycleAdd, "keep"},
				{lifecycleRemove, "ghost"},
			},
			wantContains: map[string]bool{"keep": true, "ghost": false},
			wantCount:    1,
		},
		{
			name: "full add remove cycle",
			steps: []lifecycleStep{
				{lifecycleAdd, "a"},
				{lifecycleAdd, "b"},
				{lifecycleRemove, "a"},
				{lifecycleRemove, "b"},
			},
			wantContains: map[string]bool{"a": false, "b": false},
			wantCount:    0,
			expectEmpty:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cf, err := NewCountingFromTarget(256, 0.05)
			if err != nil {
				t.Fatal(err)
			}
			applyLifecycle(t, cf, tt.steps)

			for key, want := range tt.wantContains {
				if got := cf.Contains([]byte(key)); got != want {
					t.Errorf("Contains(%q) = %v, want %v", key, got, want)
				}
			}
			if got := cf.ApproximateCount(); got != uint64(tt.wantCount) {
				t.Errorf("ApproximateCount() = %d, want %d", got, tt.wantCount)
			}
			if ratio := cf.FillRatio(); ratio < 0 || ratio > 1 {
				t.Errorf("FillRatio() = %v, want in [0, 1]", ratio)
			}
			if tt.expectEmpty && cf.FillRatio() != 0 {
				t.Errorf("FillRatio() = %v, want 0", cf.FillRatio())
			}
		})
	}
}

func TestCountingFilterLifecycleProperty(t *testing.T) {
	prop := func(seed uint8, ops uint8) bool {
		if ops == 0 {
			return true
		}
		cf, err := NewCountingFromTarget(128, 0.1)
		if err != nil {
			return false
		}
		keys := []string{"a", "b", "c"}
		netAdds := 0
		for i := uint8(0); i < ops && i < 40; i++ {
			keyIdx := int((seed + i) % 3)
			key := []byte(keys[keyIdx])
			if (seed+i)%2 == 0 {
				if err := cf.Add(key); err != nil {
					return false
				}
				netAdds++
			} else {
				if cf.Remove(key) {
					netAdds--
				}
			}
			ratio := cf.FillRatio()
			if ratio < 0 || ratio > 1 {
				return false
			}
			if int(cf.ApproximateCount()) != netAdds {
				return false
			}
		}
		return true
	}
	cfg := &quick.Config{MaxCount: 200}
	if err := quick.Check(prop, cfg); err != nil {
		t.Error(err)
	}
}

func TestCountingFilterRemoveAbsentProperty(t *testing.T) {
	prop := func(capacity uint16) bool {
		if capacity == 0 {
			capacity = 1
		}
		cf, err := NewCountingFromTarget(uint64(capacity), 0.05)
		if err != nil {
			return false
		}
		before := cf.ApproximateCount()
		if cf.Remove([]byte("never-seen")) {
			return false
		}
		if cf.ApproximateCount() != before {
			return false
		}
		if cf.FillRatio() != 0 {
			return false
		}
		return true
	}
	cfg := &quick.Config{MaxCount: 100}
	if err := quick.Check(prop, cfg); err != nil {
		t.Error(err)
	}
}

func TestCountingFilterFillRatioProperty(t *testing.T) {
	prop := func(bits uint16, hashCount uint8) bool {
		if bits < 8 {
			bits = 8
		}
		if hashCount == 0 {
			hashCount = 1
		}
		cf, err := NewCounting(uint64(bits), uint(hashCount))
		if err != nil {
			return false
		}
		for i := 0; i < 20; i++ {
			key := []byte{byte(i)}
			if err := cf.Add(key); err != nil {
				return false
			}
			ratio := cf.FillRatio()
			if ratio <= 0 || ratio > 1 {
				return false
			}
		}
		return true
	}
	cfg := &quick.Config{MaxCount: 100}
	if err := quick.Check(prop, cfg); err != nil {
		t.Error(err)
	}
}
