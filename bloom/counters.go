package bloom

import "fmt"

const (
	counterMax8  = uint8(255)
	counterMax16 = uint16(65535)
	counterMax32 = uint32(4294967295)
	counterMax64 = uint64(18446744073709551615)
)

type counterStore interface {
	at(idx uint64) uint64
	inc(idx uint64) error
	dec(idx uint64)
	clear()
	max() uint64
	occupied() uint64
	bytesPerCounter() uint64
	limit() uint64
}

type counterStore8 struct {
	counters []uint8
}

func newCounterStore8(m uint64) counterStore8 {
	return counterStore8{counters: make([]uint8, m)}
}

func (s counterStore8) at(idx uint64) uint64 { return uint64(s.counters[idx]) }

func (s counterStore8) inc(idx uint64) error {
	if s.counters[idx] == counterMax8 {
		return ErrCounterOverflow
	}
	s.counters[idx]++
	return nil
}

func (s counterStore8) dec(idx uint64) {
	if s.counters[idx] > 0 {
		s.counters[idx]--
	}
}

func (s counterStore8) clear() {
	for i := range s.counters {
		s.counters[i] = 0
	}
}

func (s counterStore8) max() uint64 {
	var max uint64
	for _, c := range s.counters {
		if v := uint64(c); v > max {
			max = v
		}
	}
	return max
}

func (s counterStore8) occupied() uint64 {
	var occupied uint64
	for _, c := range s.counters {
		if c > 0 {
			occupied++
		}
	}
	return occupied
}

func (s counterStore8) bytesPerCounter() uint64 { return 1 }

func (s counterStore8) limit() uint64 { return uint64(counterMax8) }

type counterStore16 struct {
	counters []uint16
}

func newCounterStore16(m uint64) counterStore16 {
	return counterStore16{counters: make([]uint16, m)}
}

func (s counterStore16) at(idx uint64) uint64 { return uint64(s.counters[idx]) }

func (s counterStore16) inc(idx uint64) error {
	if s.counters[idx] == counterMax16 {
		return ErrCounterOverflow
	}
	s.counters[idx]++
	return nil
}

func (s counterStore16) dec(idx uint64) {
	if s.counters[idx] > 0 {
		s.counters[idx]--
	}
}

func (s counterStore16) clear() {
	for i := range s.counters {
		s.counters[i] = 0
	}
}

func (s counterStore16) max() uint64 {
	var max uint64
	for _, c := range s.counters {
		if v := uint64(c); v > max {
			max = v
		}
	}
	return max
}

func (s counterStore16) occupied() uint64 {
	var occupied uint64
	for _, c := range s.counters {
		if c > 0 {
			occupied++
		}
	}
	return occupied
}

func (s counterStore16) bytesPerCounter() uint64 { return 2 }

func (s counterStore16) limit() uint64 { return uint64(counterMax16) }

type counterStore32 struct {
	counters []uint32
}

func newCounterStore32(m uint64) counterStore32 {
	return counterStore32{counters: make([]uint32, m)}
}

func (s counterStore32) at(idx uint64) uint64 { return uint64(s.counters[idx]) }

func (s counterStore32) inc(idx uint64) error {
	if s.counters[idx] == counterMax32 {
		return ErrCounterOverflow
	}
	s.counters[idx]++
	return nil
}

func (s counterStore32) dec(idx uint64) {
	if s.counters[idx] > 0 {
		s.counters[idx]--
	}
}

func (s counterStore32) clear() {
	for i := range s.counters {
		s.counters[i] = 0
	}
}

func (s counterStore32) max() uint64 {
	var max uint64
	for _, c := range s.counters {
		if v := uint64(c); v > max {
			max = v
		}
	}
	return max
}

func (s counterStore32) occupied() uint64 {
	var occupied uint64
	for _, c := range s.counters {
		if c > 0 {
			occupied++
		}
	}
	return occupied
}

func (s counterStore32) bytesPerCounter() uint64 { return 4 }

func (s counterStore32) limit() uint64 { return uint64(counterMax32) }

type counterStore64 struct {
	counters []uint64
}

func newCounterStore64(m uint64) counterStore64 {
	return counterStore64{counters: make([]uint64, m)}
}

func (s counterStore64) at(idx uint64) uint64 { return s.counters[idx] }

func (s counterStore64) inc(idx uint64) error {
	if s.counters[idx] == counterMax64 {
		return ErrCounterOverflow
	}
	s.counters[idx]++
	return nil
}

func (s counterStore64) dec(idx uint64) {
	if s.counters[idx] > 0 {
		s.counters[idx]--
	}
}

func (s counterStore64) clear() {
	for i := range s.counters {
		s.counters[i] = 0
	}
}

func (s counterStore64) max() uint64 {
	var max uint64
	for _, c := range s.counters {
		if c > max {
			max = c
		}
	}
	return max
}

func (s counterStore64) occupied() uint64 {
	var occupied uint64
	for _, c := range s.counters {
		if c > 0 {
			occupied++
		}
	}
	return occupied
}

func (s counterStore64) bytesPerCounter() uint64 { return 8 }

func (s counterStore64) limit() uint64 { return counterMax64 }

func newCounterStore(m uint64, width uint8) (counterStore, error) {
	switch width {
	case 8:
		return newCounterStore8(m), nil
	case 16:
		return newCounterStore16(m), nil
	case 32:
		return newCounterStore32(m), nil
	case 64:
		return newCounterStore64(m), nil
	default:
		return nil, fmt.Errorf("%w: %d", ErrInvalidCounterWidth, width)
	}
}
