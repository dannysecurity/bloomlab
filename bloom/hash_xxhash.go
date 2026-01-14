package bloom

import (
	"encoding/binary"
	"math/bits"
)

type xxhashHasher struct {
	seed uint64
}

func (x xxhashHasher) Strategy() Strategy { return HashXXHash }

func (x xxhashHasher) Derive(key []byte) (h1, h2 uint64) {
	return splitDoubleHashPair(xxhash128(key, x.seed))
}

// xxhash64 implements the 64-bit variant of xxHash (v1).
func xxhash64(data []byte, seed uint64) uint64 {
	const (
		prime1 = uint64(0x9e3779b185ebca87)
		prime2 = uint64(0xc2b2ae3d27d4eb4f)
		prime3 = uint64(0x165667b19e3779f9)
		prime4 = uint64(0x85ebca77c2b2ae63)
		prime5 = uint64(0x27d4eb2f165667c5)
	)

	n := len(data)
	var h uint64

	if n >= 32 {
		v1 := seed + prime1 + prime2
		v2 := seed + prime2
		v3 := seed
		v4 := seed - prime1

		for i := 0; i < n/32; i++ {
			off := i * 32
			v1 = xxhash64Round(v1, binary.LittleEndian.Uint64(data[off:]))
			v2 = xxhash64Round(v2, binary.LittleEndian.Uint64(data[off+8:]))
			v3 = xxhash64Round(v3, binary.LittleEndian.Uint64(data[off+16:]))
			v4 = xxhash64Round(v4, binary.LittleEndian.Uint64(data[off+24:]))
		}

		h = bits.RotateLeft64(v1, 1) + bits.RotateLeft64(v2, 7) +
			bits.RotateLeft64(v3, 12) + bits.RotateLeft64(v4, 18)
		h = xxhash64MergeRound(h, v1)
		h = xxhash64MergeRound(h, v2)
		h = xxhash64MergeRound(h, v3)
		h = xxhash64MergeRound(h, v4)
	} else {
		h = seed + prime5
	}

	h += uint64(n)

	off := (n / 32) * 32
	for off+8 <= n {
		k := xxhash64Round(0, binary.LittleEndian.Uint64(data[off:]))
		h ^= k
		h = bits.RotateLeft64(h, 27)*prime1 + prime4
		off += 8
	}

	if off+4 <= n {
		h ^= uint64(binary.LittleEndian.Uint32(data[off:])) * prime1
		h = bits.RotateLeft64(h, 23)*prime2 + prime3
		off += 4
	}

	for off < n {
		h ^= uint64(data[off]) * prime5
		h = bits.RotateLeft64(h, 11) * prime1
		off++
	}

	h ^= h >> 33
	h *= prime2
	h ^= h >> 29
	h *= prime3
	h ^= h >> 32
	return h
}

// xxhash128 derives two independent 64-bit hashes from one pass over key material.
// The body is processed once with paired accumulators; each half is finalized separately.
func xxhash128(data []byte, seed uint64) (h1, h2 uint64) {
	const (
		prime1 = uint64(0x9e3779b185ebca87)
		prime2 = uint64(0xc2b2ae3d27d4eb4f)
		prime3 = uint64(0x165667b19e3779f9)
		prime4 = uint64(0x85ebca77c2b2ae63)
		prime5 = uint64(0x27d4eb2f165667c5)
	)

	seed2 := seed ^ doubleHashSeedMix
	n := len(data)

	var acc1, acc2 uint64
	if n >= 32 {
		v1a, v2a, v3a, v4a := seed+prime1+prime2, seed+prime2, seed, seed-prime1
		v1b, v2b, v3b, v4b := seed2+prime1+prime2, seed2+prime2, seed2, seed2-prime1

		for i := 0; i < n/32; i++ {
			off := i * 32
			b0 := binary.LittleEndian.Uint64(data[off:])
			b1 := binary.LittleEndian.Uint64(data[off+8:])
			b2 := binary.LittleEndian.Uint64(data[off+16:])
			b3 := binary.LittleEndian.Uint64(data[off+24:])

			v1a = xxhash64Round(v1a, b0)
			v2a = xxhash64Round(v2a, b1)
			v3a = xxhash64Round(v3a, b2)
			v4a = xxhash64Round(v4a, b3)

			v1b = xxhash64Round(v1b, b0)
			v2b = xxhash64Round(v2b, b1)
			v3b = xxhash64Round(v3b, b2)
			v4b = xxhash64Round(v4b, b3)
		}

		acc1 = bits.RotateLeft64(v1a, 1) + bits.RotateLeft64(v2a, 7) +
			bits.RotateLeft64(v3a, 12) + bits.RotateLeft64(v4a, 18)
		acc1 = xxhash64MergeRound(acc1, v1a)
		acc1 = xxhash64MergeRound(acc1, v2a)
		acc1 = xxhash64MergeRound(acc1, v3a)
		acc1 = xxhash64MergeRound(acc1, v4a)

		acc2 = bits.RotateLeft64(v1b, 1) + bits.RotateLeft64(v2b, 7) +
			bits.RotateLeft64(v3b, 12) + bits.RotateLeft64(v4b, 18)
		acc2 = xxhash64MergeRound(acc2, v1b)
		acc2 = xxhash64MergeRound(acc2, v2b)
		acc2 = xxhash64MergeRound(acc2, v3b)
		acc2 = xxhash64MergeRound(acc2, v4b)
	} else {
		acc1 = seed + prime5
		acc2 = seed2 + prime5
	}

	acc1 += uint64(n)
	acc2 += uint64(n)

	off := (n / 32) * 32
	for off+8 <= n {
		k := xxhash64Round(0, binary.LittleEndian.Uint64(data[off:]))
		acc1 ^= k
		acc1 = bits.RotateLeft64(acc1, 27)*prime1 + prime4
		acc2 ^= k * 0x9ddfea08eb382d69
		acc2 = bits.RotateLeft64(acc2, 27)*prime1 + prime4
		off += 8
	}

	if off+4 <= n {
		word := uint64(binary.LittleEndian.Uint32(data[off:]))
		acc1 ^= word * prime1
		acc1 = bits.RotateLeft64(acc1, 23)*prime2 + prime3
		acc2 ^= word * (prime1 ^ 0x85ebca6b)
		acc2 = bits.RotateLeft64(acc2, 23)*prime2 + prime3
		off += 4
	}

	for off < n {
		b := uint64(data[off])
		acc1 ^= b * prime5
		acc1 = bits.RotateLeft64(acc1, 11) * prime1
		acc2 ^= b * (prime5 ^ 0x27d4eb2f)
		acc2 = bits.RotateLeft64(acc2, 11) * prime1
		off++
	}

	h1 = xxhash64Finalize(acc1)
	h2 = xxhash64Finalize(acc2)
	return h1, h2
}

func xxhash64Finalize(h uint64) uint64 {
	const (
		prime2 = uint64(0xc2b2ae3d27d4eb4f)
		prime3 = uint64(0x165667b19e3779f9)
	)
	h ^= h >> 33
	h *= prime2
	h ^= h >> 29
	h *= prime3
	h ^= h >> 32
	return h
}

func xxhash64Round(acc, input uint64) uint64 {
	acc += input * 0xc2b2ae3d27d4eb4f
	acc = bits.RotateLeft64(acc, 31)
	acc *= 0x9e3779b185ebca87
	return acc
}

func xxhash64MergeRound(acc, val uint64) uint64 {
	val = xxhash64Round(0, val)
	acc ^= val
	acc = acc*0x9e3779b185ebca87 + 0x85ebca77c2b2ae63
	return acc
}
