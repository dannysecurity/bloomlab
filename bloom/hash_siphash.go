package bloom

import (
	"encoding/binary"
	"math/bits"
)

type siphashHasher struct {
	seed uint64
}

func (s siphashHasher) Strategy() Strategy { return HashSipHash }

func (s siphashHasher) Derive(key []byte) (h1, h2 uint64) {
	return splitDoubleHashPair(siphash128(key, s.seed))
}

const (
	sipC0 = uint64(0x736f6d6570736575)
	sipC1 = uint64(0x646f72616e646f6d)
	sipC2 = uint64(0x6c7967656e657261)
	sipC3 = uint64(0x7465646279746573)
)

func siphashInit(k0, k1 uint64) (v0, v1, v2, v3 uint64) {
	return k0 ^ sipC0, k1 ^ sipC1, k0 ^ sipC2, k1 ^ sipC3
}

func sipRound(v0, v1, v2, v3 *uint64) {
	*v0 += *v1
	*v1 = bits.RotateLeft64(*v1, 13)
	*v1 ^= *v0
	*v0 = bits.RotateLeft64(*v0, 32)

	*v2 += *v3
	*v3 = bits.RotateLeft64(*v3, 16)
	*v3 ^= *v2

	*v0 += *v3
	*v3 = bits.RotateLeft64(*v3, 21)
	*v3 ^= *v0

	*v2 += *v1
	*v1 = bits.RotateLeft64(*v1, 17)
	*v1 ^= *v2
	*v2 = bits.RotateLeft64(*v2, 32)
}

// siphash128 derives two 64-bit hashes in one pass using paired SipHash-2-4 state.
func siphash128(data []byte, seed uint64) (h1, h2 uint64) {
	k0a, k1a := pairDerivedSeeds(seed)
	k0b, k1b := pairDerivedSeeds(splitMix64(seed))

	v0a, v1a, v2a, v3a := siphashInit(k0a, k1a)
	v0b, v1b, v2b, v3b := siphashInit(k0b, k1b)

	end := len(data) - len(data)%8
	for off := 0; off < end; off += 8 {
		m := binary.LittleEndian.Uint64(data[off:])
		v3a ^= m
		v3b ^= m
		sipRound(&v0a, &v1a, &v2a, &v3a)
		sipRound(&v0b, &v1b, &v2b, &v3b)
		sipRound(&v0a, &v1a, &v2a, &v3a)
		sipRound(&v0b, &v1b, &v2b, &v3b)
		v0a ^= m
		v0b ^= m
	}

	var b uint64
	rem := len(data) - end
	switch rem {
	case 7:
		b |= uint64(data[end+6]) << 48
		fallthrough
	case 6:
		b |= uint64(data[end+5]) << 40
		fallthrough
	case 5:
		b |= uint64(data[end+4]) << 32
		fallthrough
	case 4:
		b |= uint64(data[end+3]) << 24
		fallthrough
	case 3:
		b |= uint64(data[end+2]) << 16
		fallthrough
	case 2:
		b |= uint64(data[end+1]) << 8
		fallthrough
	case 1:
		b |= uint64(data[end])
	}
	b |= uint64(len(data)) << 56

	v3a ^= b
	v3b ^= b
	sipRound(&v0a, &v1a, &v2a, &v3a)
	sipRound(&v0b, &v1b, &v2b, &v3b)
	sipRound(&v0a, &v1a, &v2a, &v3a)
	sipRound(&v0b, &v1b, &v2b, &v3b)
	v0a ^= b
	v0b ^= b

	v2a ^= 0xff
	v2b ^= 0xff
	for i := 0; i < 4; i++ {
		sipRound(&v0a, &v1a, &v2a, &v3a)
		sipRound(&v0b, &v1b, &v2b, &v3b)
	}
	return v0a ^ v1a ^ v2a ^ v3a, v0b ^ v1b ^ v2b ^ v3b
}
