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
	const mix = uint64(0x9e3779b97f4a7c15)
	h1 = xxhash64(key, x.seed)
	h2 = ensureH2NonZero(xxhash64(key, x.seed^mix))
	return h1, h2
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
