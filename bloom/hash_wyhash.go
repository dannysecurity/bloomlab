package bloom

import (
	"encoding/binary"
	"math/bits"
)

type wyhashHasher struct {
	seed uint64
}

func (w wyhashHasher) Strategy() Strategy { return HashWyhash }

func (w wyhashHasher) Derive(key []byte) (h1, h2 uint64) {
	seed1, seed2 := pairDerivedSeeds(w.seed)
	h1 = wyhash64(key, seed1)
	h2 = ensureH2NonZero(wyhash64(key, seed2))
	return h1, h2
}

// wyhash64 implements wyhash final version 1 (64-bit output).
// Reference: https://github.com/wangyi-fudan/wyhash
func wyhash64(data []byte, seed uint64) uint64 {
	const (
		s1 = uint64(0xe7037ed1a0b428db)
		s2 = uint64(0x8ebc6af09c88c6e3)
		s3 = uint64(0x589965cc75374cc3)
		s4 = uint64(0x1d8e4e27c47d124f)
	)

	length := len(data)
	i := length
	off := 0

	if i > 64 {
		see1 := seed
		for i > 64 {
			seed = wymix(wyReadU64(data, off)^s1, wyReadU64(data, off+8)^seed) ^
				wymix(wyReadU64(data, off+16)^s2, wyReadU64(data, off+24)^seed)
			see1 = wymix(wyReadU64(data, off+32)^s3, wyReadU64(data, off+40)^see1) ^
				wymix(wyReadU64(data, off+48)^s4, wyReadU64(data, off+56)^see1)
			off += 64
			i -= 64
		}
		seed ^= see1
	}

	for i > 16 {
		seed = wymix(wyReadU64(data, off)^s1, wyReadU64(data, off+8)^seed)
		off += 16
		i -= 16
	}

	switch {
	case i == 0:
		return wymix(s1, wymix(s1, seed))
	case i < 4:
		a := uint64(data[off])<<16 | uint64(data[off+i/2])<<8 | uint64(data[off+i-1])
		return wymix(s1^uint64(length), wymix(a^s1, seed))
	case i == 4:
		a := wyReadU32(data, off)
		return wymix(s1^uint64(length), wymix(a^s1, seed))
	case i < 8:
		a := wyReadU32(data, off)
		b := wyReadU32(data, off+i-4)
		return wymix(s1^uint64(length), wymix(a^s1, b^seed))
	case i == 8:
		a := wyReadU64(data, off)
		return wymix(s1^uint64(length), wymix(a^s1, seed))
	default:
		a := wyReadU64(data, off)
		b := wyReadU64(data, off+i-8)
		return wymix(s1^uint64(length), wymix(a^s1, b^seed))
	}
}

func wymix(a, b uint64) uint64 {
	hi, lo := bits.Mul64(a, b)
	return hi ^ lo
}

func wyReadU32(data []byte, off int) uint64 {
	var buf [4]byte
	copy(buf[:], data[off:])
	return uint64(binary.LittleEndian.Uint32(buf[:]))
}

func wyReadU64(data []byte, off int) uint64 {
	if off+8 <= len(data) {
		return binary.LittleEndian.Uint64(data[off:])
	}
	var buf [8]byte
	copy(buf[:], data[off:])
	return binary.LittleEndian.Uint64(buf[:])
}
