package bloom

import (
	"testing"
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
	if _, err := New(0, 0.01); err != ErrInvalidCapacity {
		t.Errorf("expected ErrInvalidCapacity, got %v", err)
	}
	if _, err := New(100, 0); err != ErrInvalidFPR {
		t.Errorf("expected ErrInvalidFPR, got %v", err)
	}
	if _, err := New(100, 1.0); err != ErrInvalidFPR {
		t.Errorf("expected ErrInvalidFPR, got %v", err)
	}
}

func TestFilterSizing(t *testing.T) {
	f, err := New(10_000, 0.001)
	if err != nil {
		t.Fatal(err)
	}
	if f.BitCount() < 64 {
		t.Errorf("bit count too small: %d", f.BitCount())
	}
	if f.HashCount() < 1 {
		t.Error("hash count must be at least 1")
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
