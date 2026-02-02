package bloom

import "testing"

func TestCounterStore4PackedNibbles(t *testing.T) {
	store, err := newCounterStore(5, 4)
	if err != nil {
		t.Fatal(err)
	}
	s := store.(counterStore4)

	// Odd m uses high nibble of the last byte; low nibble stays zero.
	if got := s.storageBytes(); got != 3 {
		t.Fatalf("storageBytes() = %d, want 3", got)
	}

	for idx := uint64(0); idx < 5; idx++ {
		if err := store.inc(idx); err != nil {
			t.Fatalf("inc(%d): %v", idx, err)
		}
	}
	if got := store.occupied(); got != 5 {
		t.Fatalf("occupied() = %d, want 5", got)
	}
	if got := s.readNibble(4); got != 1 {
		t.Fatalf("counter 4 = %d, want 1", got)
	}
	if got := s.readNibble(5); got != 0 {
		t.Fatalf("unused nibble = %d, want 0", got)
	}

	store.clear()
	if got := store.max(); got != 0 {
		t.Fatalf("after clear max() = %d, want 0", got)
	}
}

func TestCounterStore4AdjacentCountersIndependent(t *testing.T) {
	store, err := newCounterStore(4, 4)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 15; i++ {
		if err := store.inc(0); err != nil {
			t.Fatalf("inc high nibble %d: %v", i, err)
		}
	}
	if err := store.inc(0); err != ErrCounterOverflow {
		t.Fatalf("expected overflow on nibble 0, got %v", err)
	}
	if got := store.at(1); got != 0 {
		t.Fatalf("adjacent nibble changed: at(1) = %d, want 0", got)
	}

	if err := store.inc(1); err != nil {
		t.Fatal(err)
	}
	if got := store.at(0); got != 15 {
		t.Fatalf("overflow on nibble 1 changed nibble 0: at(0) = %d, want 15", got)
	}
}
