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
	return splitDoubleHashPair(murmur3_128(key, m.seed))
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

// murmur3_128 implements MurmurHash3 x64_128 in one pass over key material.
func murmur3_128(data []byte, seed uint64) (h1, h2 uint64) {
	const (
		c1 = uint64(0x87c37b91114253d5)
		c2 = uint64(0x4cf5ad432745937f)
	)

	h1, h2 = seed, seed
	nblocks := len(data) / 16

	for i := 0; i < nblocks; i++ {
		k1 := binary.LittleEndian.Uint64(data[i*16:])
		k2 := binary.LittleEndian.Uint64(data[i*16+8:])

		k1 *= c1
		k1 = bits.RotateLeft64(k1, 31)
		k1 *= c2
		h1 ^= k1

		h1 = bits.RotateLeft64(h1, 27)
		h1 += h2
		h1 = h1*5 + 0x52dce729

		k2 *= c2
		k2 = bits.RotateLeft64(k2, 33)
		k2 *= c1
		h2 ^= k2

		h2 = bits.RotateLeft64(h2, 31)
		h2 += h1
		h2 = h2*5 + 0x38495ab5
	}

	tail := data[nblocks*16:]
	var k1, k2 uint64
	switch len(tail) {
	case 15:
		k2 ^= uint64(tail[14]) << 48
		fallthrough
	case 14:
		k2 ^= uint64(tail[13]) << 40
		fallthrough
	case 13:
		k2 ^= uint64(tail[12]) << 32
		fallthrough
	case 12:
		k2 ^= uint64(tail[11]) << 24
		fallthrough
	case 11:
		k2 ^= uint64(tail[10]) << 16
		fallthrough
	case 10:
		k2 ^= uint64(tail[9]) << 8
		fallthrough
	case 9:
		k2 ^= uint64(tail[8])
		fallthrough
	case 8:
		k1 ^= uint64(tail[7]) << 56
		fallthrough
	case 7:
		k1 ^= uint64(tail[6]) << 48
		fallthrough
	case 6:
		k1 ^= uint64(tail[5]) << 40
		fallthrough
	case 5:
		k1 ^= uint64(tail[4]) << 32
		fallthrough
	case 4:
		k1 ^= uint64(tail[3]) << 24
		fallthrough
	case 3:
		k1 ^= uint64(tail[2]) << 16
		fallthrough
	case 2:
		k1 ^= uint64(tail[1]) << 8
		fallthrough
	case 1:
		k1 ^= uint64(tail[0])
	}

	k1 *= c1
	k1 = bits.RotateLeft64(k1, 31)
	k1 *= c2
	h1 ^= k1

	k2 *= c2
	k2 = bits.RotateLeft64(k2, 33)
	k2 *= c1
	h2 ^= k2

	h1 ^= uint64(len(data))
	h2 ^= uint64(len(data))

	h1 += h2
	h2 += h1

	h1 = murmur3_fmix64(h1)
	h2 = murmur3_fmix64(h2)

	h1 += h2
	h2 += h1
	return h1, h2
}

func murmur3_fmix64(k uint64) uint64 {
	k ^= k >> 33
	k *= 0xff51afd7ed558ccd
	k ^= k >> 33
	k *= 0xc4ceb9fe1a85ec53
	k ^= k >> 33
	return k
}
