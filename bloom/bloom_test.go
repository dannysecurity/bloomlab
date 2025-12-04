package bloom

import (
	"testing"
	"testing/quick"
)

func TestFilterAddContains(t *testing.T) {
	f, err := New(1000, 0.01)
	if err != nil {
		t.Fatal(err)
	}

	keys := [][]byte{
		[]byte("alpha"),
		[]byte("beta"),
		[]byte("gamma"),
	}

	for _, k := range keys {
		f.Add(k)
	}

	for _, k := range keys {
		if !f.Contains(k) {
			t.Errorf("expected %q to be present", k)
		}
	}

	if f.Contains([]byte("missing")) {
		t.Error("unexpected false positive for missing key (possible but unlikely in small test)")
	}
}

func TestFilterInvalidParams(t *testing.T) {
	tests := []struct {
		name     string
		capacity uint64
		fpr      float64
		wantErr  error
	}{
		{"zero capacity", 0, 0.01, ErrInvalidCapacity},
		{"zero fpr", 100, 0, ErrInvalidFPR},
		{"fpr equals one", 100, 1.0, ErrInvalidFPR},
		{"negative fpr", 100, -0.01, ErrInvalidFPR},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.capacity, tt.fpr)
			if err != tt.wantErr {
				t.Errorf("New(%d, %v) error = %v, want %v", tt.capacity, tt.fpr, err, tt.wantErr)
			}
		})
	}
}

func TestFilterSizing(t *testing.T) {
	tests := []struct {
		name     string
		capacity uint64
		fpr      float64
		minBits  uint64
		minK     uint
	}{
		{"large tight filter", 10_000, 0.001, 64, 1},
		{"small loose filter", 100, 0.1, 64, 1},
		{"medium default", 1_000, 0.01, 64, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := New(tt.capacity, tt.fpr)
			if err != nil {
				t.Fatal(err)
			}
			if f.BitCount() < tt.minBits {
				t.Errorf("BitCount() = %d, want >= %d", f.BitCount(), tt.minBits)
			}
			if f.HashCount() < tt.minK {
				t.Errorf("HashCount() = %d, want >= %d", f.HashCount(), tt.minK)
			}
		})
	}
}

func TestFilterAddContainsProperty(t *testing.T) {
	prop := func(capacity uint16, key []byte) bool {
		if capacity == 0 {
			capacity = 1
		}
		if len(key) == 0 {
			return true
		}
		f, err := New(uint64(capacity), 0.05)
		if err != nil {
			return false
		}
		f.Add(key)
		return f.Contains(key)
	}
	cfg := &quick.Config{MaxCount: 200}
	if err := quick.Check(prop, cfg); err != nil {
		t.Error(err)
	}
}

func TestFalsePositiveRate(t *testing.T) {
	const n = 5000
	f, err := New(n, 0.01)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < n; i++ {
		f.Add([]byte{byte(i >> 8), byte(i)})
	}

	falsePositives := 0
	const trials = 5000
	for i := n; i < n+trials; i++ {
		if f.Contains([]byte{byte(i >> 8), byte(i)}) {
			falsePositives++
		}
	}

	rate := float64(falsePositives) / trials
	if rate > 0.05 {
		t.Errorf("false positive rate %.4f exceeds tolerance for p=0.01", rate)
	}
}
