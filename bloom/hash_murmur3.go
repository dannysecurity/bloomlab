package bloom

import (
	"encoding/binary"
	"math/bits"
)

type murmur3Hasher struct {
	seed uint64
}

func (m murmur3Hasher) Strategy() Strategy { return HashMurmur3 }

func (m murmur3Hasher) Derive(key []byte) (h1, h2 uint64) {
	const mix = uint64(0x9e3779b97f4a7c15)
	h1 = murmur3_64(key, m.seed)
	h2 = ensureH2NonZero(murmur3_64(key, m.seed^mix))
	return h1, h2
}

// murmur3_64 implements the 64-bit finalizer from MurmurHash3 (x64_128 body).
func murmur3_64(data []byte, seed uint64) uint64 {
	const (
		c1 = uint64(0x87c37b91114253d5)
		c2 = uint64(0x4cf5ad432745937f)
	)

	h := seed
	nblocks := len(data) / 8

	for i := 0; i < nblocks; i++ {
		k := binary.LittleEndian.Uint64(data[i*8:])
		k *= c1
		k = bits.RotateLeft64(k, 31)
		k *= c2

		h ^= k
		h = bits.RotateLeft64(h, 27)
		h = h*5 + 0x52dce729
	}

	tail := data[nblocks*8:]
	var k2 uint64
	switch len(tail) {
	case 7:
		k2 ^= uint64(tail[6]) << 48
		fallthrough
	case 6:
		k2 ^= uint64(tail[5]) << 40
		fallthrough
	case 5:
		k2 ^= uint64(tail[4]) << 32
		fallthrough
	case 4:
		k2 ^= uint64(tail[3]) << 24
		fallthrough
	case 3:
		k2 ^= uint64(tail[2]) << 16
		fallthrough
	case 2:
		k2 ^= uint64(tail[1]) << 8
		fallthrough
	case 1:
		k2 ^= uint64(tail[0])
		k2 *= c2
		k2 = bits.RotateLeft64(k2, 33)
		k2 *= c1
		h ^= k2
	}

	h ^= uint64(len(data))
	h ^= h >> 33
	h *= 0xff51afd7ed558ccd
	h ^= h >> 33
	h *= 0xc4ceb9fe1a85ec53
	h ^= h >> 33
	return h
}
