package bloom

import "fmt"

const (
	counterMax2  = uint8(3)
	counterMax4  = uint8(15)
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
	storageBytes() uint64
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

func (s counterStore8) storageBytes() uint64 { return uint64(len(s.counters)) }

func (s counterStore8) limit() uint64 { return uint64(counterMax8) }

type counterStore2 struct {
	m    uint64
	data []byte // four 2-bit counters per byte
}

func newCounterStore2(m uint64) counterStore2 {
	return counterStore2{m: m, data: make([]byte, (m+3)/4)}
}

func twoBitSlot(idx uint64) (byteIdx uint64, shift uint) {
	slot := idx % 4
	return idx / 4, uint(slot * 2)
}

func (s counterStore2) readCounter(idx uint64) uint8 {
	byteIdx, shift := twoBitSlot(idx)
	return (s.data[byteIdx] >> shift) & 0x03
}

func (s counterStore2) writeCounter(idx uint64, val uint8) {
	byteIdx, shift := twoBitSlot(idx)
	mask := byte(0x03 << shift)
	s.data[byteIdx] = (s.data[byteIdx] &^ mask) | ((val & 0x03) << shift)
}

func (s counterStore2) at(idx uint64) uint64 { return uint64(s.readCounter(idx)) }

func (s counterStore2) inc(idx uint64) error {
	v := s.readCounter(idx)
	if v == counterMax2 {
		return ErrCounterOverflow
	}
	s.writeCounter(idx, v+1)
	return nil
}

func (s counterStore2) dec(idx uint64) {
	v := s.readCounter(idx)
	if v > 0 {
		s.writeCounter(idx, v-1)
	}
}

func (s counterStore2) clear() {
	for i := range s.data {
		s.data[i] = 0
	}
}

func (s counterStore2) max() uint64 {
	var max uint64
	for idx := uint64(0); idx < s.m; idx++ {
		if v := uint64(s.readCounter(idx)); v > max {
			max = v
		}
	}
	return max
}

func (s counterStore2) occupied() uint64 {
	var occupied uint64
	for idx := uint64(0); idx < s.m; idx++ {
		if s.readCounter(idx) > 0 {
			occupied++
		}
	}
	return occupied
}

func (s counterStore2) bytesPerCounter() uint64 { return 0 }

func (s counterStore2) storageBytes() uint64 { return uint64(len(s.data)) }

func (s counterStore2) limit() uint64 { return uint64(counterMax2) }

type counterStore4 struct {
	m    uint64
	data []byte // two 4-bit counters per byte
}

func newCounterStore4(m uint64) counterStore4 {
	return counterStore4{m: m, data: make([]byte, (m+1)/2)}
}

func nibbleSlot(idx uint64) (byteIdx uint64, high bool) {
	return idx / 2, idx%2 == 0
}

func (s counterStore4) readNibble(idx uint64) uint8 {
	byteIdx, high := nibbleSlot(idx)
	if high {
		return s.data[byteIdx] >> 4
	}
	return s.data[byteIdx] & 0x0f
}

func (s counterStore4) writeNibble(idx uint64, val uint8) {
	byteIdx, high := nibbleSlot(idx)
	if high {
		s.data[byteIdx] = (s.data[byteIdx] & 0x0f) | (val << 4)
		return
	}
	s.data[byteIdx] = (s.data[byteIdx] & 0xf0) | val
}

func (s counterStore4) at(idx uint64) uint64 { return uint64(s.readNibble(idx)) }

func (s counterStore4) inc(idx uint64) error {
	v := s.readNibble(idx)
	if v == counterMax4 {
		return ErrCounterOverflow
	}
	s.writeNibble(idx, v+1)
	return nil
}

func (s counterStore4) dec(idx uint64) {
	v := s.readNibble(idx)
	if v > 0 {
		s.writeNibble(idx, v-1)
	}
}

func (s counterStore4) clear() {
	for i := range s.data {
		s.data[i] = 0
	}
}

func (s counterStore4) max() uint64 {
	var max uint64
	for idx := uint64(0); idx < s.m; idx++ {
		if v := uint64(s.readNibble(idx)); v > max {
			max = v
		}
	}
	return max
}

func (s counterStore4) occupied() uint64 {
	var occupied uint64
	for idx := uint64(0); idx < s.m; idx++ {
		if s.readNibble(idx) > 0 {
			occupied++
		}
	}
	return occupied
}

func (s counterStore4) bytesPerCounter() uint64 { return 0 }

func (s counterStore4) storageBytes() uint64 { return uint64(len(s.data)) }

func (s counterStore4) limit() uint64 { return uint64(counterMax4) }

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

func (s counterStore16) storageBytes() uint64 { return uint64(len(s.counters)) * 2 }

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

func (s counterStore32) storageBytes() uint64 { return uint64(len(s.counters)) * 4 }

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

func (s counterStore64) storageBytes() uint64 { return uint64(len(s.counters)) * 8 }

func (s counterStore64) limit() uint64 { return counterMax64 }

func newCounterStore(m uint64, width uint8) (counterStore, error) {
	switch width {
	case 2:
		return newCounterStore2(m), nil
	case 4:
		return newCounterStore4(m), nil
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
