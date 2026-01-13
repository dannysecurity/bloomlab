package bloom

import (
	"errors"
	"testing"
)

func TestCounterStoreWidths(t *testing.T) {
	tests := []struct {
		name          string
		width         uint8
		bytesPer      uint64
		limit         uint64
		overflowAfter int
	}{
		{
			name:          "8-bit",
			width:         8,
			bytesPer:      1,
			limit:         255,
			overflowAfter: 255,
		},
		{
			name:          "16-bit",
			width:         16,
			bytesPer:      2,
			limit:         65535,
			overflowAfter: 65535,
		},
		{
			name:          "32-bit",
			width:         32,
			bytesPer:      4,
			limit:         4294967295,
			overflowAfter: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, err := newCounterStore(16, tt.width)
			if err != nil {
				t.Fatal(err)
			}
			if got := store.bytesPerCounter(); got != tt.bytesPer {
				t.Fatalf("bytesPerCounter() = %d, want %d", got, tt.bytesPer)
			}
			if got := store.limit(); got != tt.limit {
				t.Fatalf("limit() = %d, want %d", got, tt.limit)
			}
			if got := store.occupied(); got != 0 {
				t.Fatalf("empty occupied() = %d, want 0", got)
			}

			for i := 0; i < tt.overflowAfter; i++ {
				if err := store.inc(0); err != nil {
					t.Fatalf("inc %d: %v", i, err)
				}
			}
			if tt.overflowAfter > 0 {
				if got := store.at(0); got != uint64(tt.overflowAfter) {
					t.Fatalf("at(0) = %d, want %d", got, tt.overflowAfter)
				}
				if got := store.max(); got != uint64(tt.overflowAfter) {
					t.Fatalf("max() = %d, want %d", got, tt.overflowAfter)
				}
				if got := store.occupied(); got != 1 {
					t.Fatalf("occupied() = %d, want 1", got)
				}
				if err := store.inc(0); err != ErrCounterOverflow {
					t.Fatalf("expected overflow, got %v", err)
				}

				store.dec(0)
				if got := store.at(0); got != uint64(tt.overflowAfter-1) {
					t.Fatalf("after dec at(0) = %d, want %d", got, tt.overflowAfter-1)
				}
			} else {
				const sample = 512
				for i := 0; i < sample; i++ {
					if err := store.inc(0); err != nil {
						t.Fatalf("inc %d: %v", i, err)
					}
				}
				if got := store.at(0); got != sample {
					t.Fatalf("at(0) = %d, want %d", got, sample)
				}
				store.dec(0)
				if got := store.at(0); got != sample-1 {
					t.Fatalf("after dec at(0) = %d, want %d", got, sample-1)
				}
			}

			store.clear()
			if got := store.max(); got != 0 {
				t.Fatalf("after clear max() = %d, want 0", got)
			}
		})
	}
}

func TestNewCounterStoreInvalidWidth(t *testing.T) {
	_, err := newCounterStore(8, 24)
	if !errors.Is(err, ErrInvalidCounterWidth) {
		t.Fatalf("error = %v, want %v", err, ErrInvalidCounterWidth)
	}
}
