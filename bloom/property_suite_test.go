package bloom

import (
	"fmt"
	"math"
	"testing"
	"testing/quick"
)

// propertyQuick is the shared quick.Check budget for mathematical invariants.
var propertyQuick = &quick.Config{MaxCount: 300}

func TestTheoryFPRRangeProperty(t *testing.T) {
	prop := func(n, m uint32, k uint8) bool {
		if m == 0 || k == 0 || n == 0 {
			return TheoryFalsePositiveRate(uint64(n), uint64(m), uint(k)) == 0
		}
		p := TheoryFalsePositiveRate(uint64(n), uint64(m), uint(k))
		return p >= 0 && p <= 1
	}
	if err := quick.Check(prop, propertyQuick); err != nil {
		t.Error(err)
	}
}

func TestTheoryFPRMonotonicInNProperty(t *testing.T) {
	prop := func(m uint32, k uint8, nLo, nHi uint32) bool {
		if m == 0 || k == 0 {
			return true
		}
		if nLo > nHi {
			nLo, nHi = nHi, nLo
		}
		if nLo == 0 {
			nLo = 1
		}
		pLo := TheoryFalsePositiveRate(uint64(nLo), uint64(m), uint(k))
		pHi := TheoryFalsePositiveRate(uint64(nHi), uint64(m), uint(k))
		return pLo <= pHi
	}
	if err := quick.Check(prop, propertyQuick); err != nil {
		t.Error(err)
	}
}

func TestTheoryFPRDecreasingInMProperty(t *testing.T) {
	prop := func(n uint32, k uint8, mLo, mHi uint32) bool {
		if n == 0 || k == 0 {
			return true
		}
		if mLo > mHi {
			mLo, mHi = mHi, mLo
		}
		if mLo == 0 {
			mLo = 1
		}
		pLo := TheoryFalsePositiveRate(uint64(n), uint64(mLo), uint(k))
		pHi := TheoryFalsePositiveRate(uint64(n), uint64(mHi), uint(k))
		return pLo >= pHi
	}
	if err := quick.Check(prop, propertyQuick); err != nil {
		t.Error(err)
	}
}

func TestTheoryFillFractionRangeProperty(t *testing.T) {
	prop := func(n, m uint32, k uint8) bool {
		if m == 0 || k == 0 || n == 0 {
			return TheoryFillFraction(uint64(n), uint64(m), uint(k)) == 0
		}
		fill := TheoryFillFraction(uint64(n), uint64(m), uint(k))
		return fill >= 0 && fill <= 1
	}
	if err := quick.Check(prop, propertyQuick); err != nil {
		t.Error(err)
	}
}

func TestTheoryFillFractionMonotonicInNProperty(t *testing.T) {
	prop := func(m uint32, k uint8, nLo, nHi uint32) bool {
		if m == 0 || k == 0 {
			return true
		}
		if nLo > nHi {
			nLo, nHi = nHi, nLo
		}
		fillLo := TheoryFillFraction(uint64(nLo), uint64(m), uint(k))
		fillHi := TheoryFillFraction(uint64(nHi), uint64(m), uint(k))
		return fillLo <= fillHi
	}
	if err := quick.Check(prop, propertyQuick); err != nil {
		t.Error(err)
	}
}

func TestTheoryFillFractionDecreasingInMProperty(t *testing.T) {
	prop := func(n uint32, k uint8, mLo, mHi uint32) bool {
		if n == 0 || k == 0 {
			return true
		}
		if mLo > mHi {
			mLo, mHi = mHi, mLo
		}
		if mLo == 0 {
			mLo = 1
		}
		fillLo := TheoryFillFraction(uint64(n), uint64(mLo), uint(k))
		fillHi := TheoryFillFraction(uint64(n), uint64(mHi), uint(k))
		return fillLo >= fillHi
	}
	if err := quick.Check(prop, propertyQuick); err != nil {
		t.Error(err)
	}
}

func TestTheoryFPRFromFillProperty(t *testing.T) {
	prop := func(n, m uint32, k uint8) bool {
		if m == 0 || k == 0 || n == 0 {
			return true
		}
		fill := TheoryFillFraction(uint64(n), uint64(m), uint(k))
		fpr := TheoryFalsePositiveRate(uint64(n), uint64(m), uint(k))
		want := math.Pow(fill, float64(k))
		return math.Abs(fpr-want) <= 1e-12*math.Max(1, want)
	}
	if err := quick.Check(prop, propertyQuick); err != nil {
		t.Error(err)
	}
}

func TestOptimalKNearMinimumFPRProperty(t *testing.T) {
	prop := func(n uint32, m uint32) bool {
		if n == 0 || m == 0 || n > 50_000 || m > 200_000 {
			return true
		}
		kOpt := optimalK(uint64(m), uint64(n), defaultMaxHashCount)
		pOpt := TheoryFalsePositiveRate(uint64(n), uint64(m), kOpt)

		minP := math.MaxFloat64
		for k := uint(1); k <= defaultMaxHashCount; k++ {
			p := TheoryFalsePositiveRate(uint64(n), uint64(m), k)
			if p < minP {
				minP = p
			}
		}
		// Integer rounding of k can sit slightly above the continuous minimum.
		return pOpt <= minP*1.05+1e-12
	}
	if err := quick.Check(prop, propertyQuick); err != nil {
		t.Error(err)
	}
}

func TestBitIndexEquivalenceProperty(t *testing.T) {
	prop := func(h1, h2 uint32, m uint32, i uint8) bool {
		if m == 0 {
			return true
		}
		got := bitIndex(uint64(h1), uint64(h2), uint64(m), uint(i))
		want := (uint64(h1) + uint64(i)*uint64(h2)) % uint64(m)
		return got == want
	}
	if err := quick.Check(prop, propertyQuick); err != nil {
		t.Error(err)
	}
}

func TestBitIndexEquivalenceTable(t *testing.T) {
	tests := []struct {
		name string
		m    uint64
		h1   uint64
		h2   uint64
	}{
		{"power of two 64", 64, 9000, 7},
		{"power of two 1024", 1024, 9000, 7},
		{"power of two 4096", 4096, 1, 4095},
		{"prime 997", 997, 42, 13},
		{"odd 1001", 1001, 12345, 67},
		{"small 3", 3, 5, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for i := uint(0); i < 32; i++ {
				want := (tt.h1 + uint64(i)*tt.h2) % tt.m
				got := bitIndex(tt.h1, tt.h2, tt.m, i)
				if got != want {
					t.Fatalf("i=%d: bitIndex=%d, want %d (m=%d)", i, got, want, tt.m)
				}
				if got >= tt.m {
					t.Fatalf("i=%d: index %d out of range for m=%d", i, got, tt.m)
				}
			}
		})
	}
}

func TestFilterCountingParityProperty(t *testing.T) {
	strategies := AllStrategies()
	for _, strategy := range strategies {
		t.Run(strategy.String(), func(t *testing.T) {
			prop := func(keyCount uint8, seed uint16) bool {
				if keyCount == 0 || keyCount > 40 {
					return true
				}
				cfg := ExplicitConfig(512, 5, WithHash(strategy), WithSeed(uint64(seed)))
				std, err := NewFilter(cfg)
				if err != nil {
					return false
				}
				cnt, err := NewCountingFilter(cfg)
				if err != nil {
					return false
				}

				keys := make([][]byte, keyCount)
				for i := range keys {
					keys[i] = []byte(fmt.Sprintf("parity-key-%d-%d", seed, i))
				}
				for _, key := range keys {
					std.Add(key)
					if err := cnt.Add(key); err != nil {
						return false
					}
				}

				for _, key := range keys {
					if std.Contains(key) != cnt.Contains(key) {
						return false
					}
				}
				for i := 0; i < 20; i++ {
					probe := []byte(fmt.Sprintf("probe-%d-%d", seed, i))
					if std.Contains(probe) != cnt.Contains(probe) {
						return false
					}
				}
				return true
			}
			if err := quick.Check(prop, propertyQuick); err != nil {
				t.Error(err)
			}
		})
	}
}

func TestFilterFillRatioProperty(t *testing.T) {
	prop := func(adds uint8) bool {
		if adds > 80 {
			return true
		}
		f, err := NewFilter(ExplicitConfig(256, 4))
		if err != nil {
			return false
		}
		prev := f.FillRatio()
		if prev < 0 || prev > 1 {
			return false
		}
		for i := uint8(0); i < adds; i++ {
			f.Add([]byte{byte(i), byte(i >> 4)})
			ratio := f.FillRatio()
			if ratio < 0 || ratio > 1 {
				return false
			}
			if ratio < prev-1e-12 {
				return false
			}
			prev = ratio
		}
		return true
	}
	if err := quick.Check(prop, propertyQuick); err != nil {
		t.Error(err)
	}
}

func TestFilterFillRatioTable(t *testing.T) {
	tests := []struct {
		name  string
		m     uint64
		k     uint
		adds  int
		maxFR float64
	}{
		{"empty", 128, 3, 0, 0},
		{"single insert", 128, 3, 1, 0.05},
		{"many inserts", 64, 4, 50, 1.0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := NewFilter(ExplicitConfig(tt.m, tt.k))
			if err != nil {
				t.Fatal(err)
			}
			for i := 0; i < tt.adds; i++ {
				f.Add([]byte{byte(i)})
			}
			ratio := f.FillRatio()
			if ratio < 0 || ratio > tt.maxFR {
				t.Errorf("FillRatio() = %g, want in [0, %g]", ratio, tt.maxFR)
			}
			if tt.adds == 0 && ratio != 0 {
				t.Errorf("empty filter FillRatio() = %g, want 0", ratio)
			}
		})
	}
}
