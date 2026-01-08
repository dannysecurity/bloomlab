package bloom

import (
	"encoding/binary"
)

type highwayHasher struct {
	key [32]byte
}

func (h highwayHasher) Strategy() Strategy { return HashHighway }

func (h highwayHasher) Derive(key []byte) (h1, h2 uint64) {
	h1, h2 = highwayHash128(key, h.key)
	h2 = ensureH2NonZero(h2)
	return h1, h2
}

// expandSeedToHighwayKey maps a filter seed into HighwayHash's required 256-bit key.
func expandSeedToHighwayKey(seed uint64) [32]byte {
	var key [32]byte
	x := seed
	for i := 0; i < 4; i++ {
		x += doubleHashSeedMix
		z := x
		z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9
		z = (z ^ (z >> 27)) * 0x94d049bb133111eb
		z ^= z >> 31
		binary.LittleEndian.PutUint64(key[i*8:], z)
	}
	return key
}

// highwayHash128 computes a 128-bit HighwayHash digest in one pass over key material.
// The portable scalar implementation follows the reference algorithm (Apache 2.0,
// github.com/minio/highwayhash highwayhash_generic.go).
func highwayHash128(data []byte, key [32]byte) (h1, h2 uint64) {
	const blockSize = 32

	var state [16]uint64
	highwayInitialize(&state, key[:])
	if n := len(data) & ^(blockSize - 1); n > 0 {
		highwayUpdate(&state, data[:n])
		data = data[n:]
	}
	if len(data) > 0 {
		var block [blockSize]byte
		offset := copy(block[:], data)
		highwayHashBuffer(&state, &block, offset)
	}

	var out [16]byte
	highwayFinalize128(&state, out[:])
	h1 = binary.LittleEndian.Uint64(out[0:])
	h2 = binary.LittleEndian.Uint64(out[8:])
	return h1, h2
}

const (
	hwyV0   = 0
	hwyV1   = 4
	hwyMul0 = 8
	hwyMul1 = 12
)

var (
	hwyInit0 = [4]uint64{0xdbe6d5d5fe4cce2f, 0xa4093822299f31d0, 0x13198a2e03707344, 0x243f6a8885a308d3}
	hwyInit1 = [4]uint64{0x3bd39e10cb0ef593, 0xc0acf169b5f18a8c, 0xbe5466cf34e90c6c, 0x452821e638d01377}
)

func highwayInitialize(state *[16]uint64, k []byte) {
	var key [4]uint64
	key[0] = binary.LittleEndian.Uint64(k[0:])
	key[1] = binary.LittleEndian.Uint64(k[8:])
	key[2] = binary.LittleEndian.Uint64(k[16:])
	key[3] = binary.LittleEndian.Uint64(k[24:])

	copy(state[hwyMul0:], hwyInit0[:])
	copy(state[hwyMul1:], hwyInit1[:])

	for i, v := range key {
		state[hwyV0+i] = hwyInit0[i] ^ v
	}

	key[0] = key[0]>>32 | key[0]<<32
	key[1] = key[1]>>32 | key[1]<<32
	key[2] = key[2]>>32 | key[2]<<32
	key[3] = key[3]>>32 | key[3]<<32

	for i, v := range key {
		state[hwyV1+i] = hwyInit1[i] ^ v
	}
}

func highwayUpdate(state *[16]uint64, msg []byte) {
	for len(msg) >= 32 {
		m := msg[:32]

		state[hwyV1+0] += binary.LittleEndian.Uint64(m) + state[hwyMul0+0]
		state[hwyMul0+0] ^= uint64(uint32(state[hwyV1+0])) * (state[hwyV0+0] >> 32)
		state[hwyV0+0] += state[hwyMul1+0]
		state[hwyMul1+0] ^= uint64(uint32(state[hwyV0+0])) * (state[hwyV1+0] >> 32)

		state[hwyV1+1] += binary.LittleEndian.Uint64(m[8:]) + state[hwyMul0+1]
		state[hwyMul0+1] ^= uint64(uint32(state[hwyV1+1])) * (state[hwyV0+1] >> 32)
		state[hwyV0+1] += state[hwyMul1+1]
		state[hwyMul1+1] ^= uint64(uint32(state[hwyV0+1])) * (state[hwyV1+1] >> 32)

		state[hwyV1+2] += binary.LittleEndian.Uint64(m[16:]) + state[hwyMul0+2]
		state[hwyMul0+2] ^= uint64(uint32(state[hwyV1+2])) * (state[hwyV0+2] >> 32)
		state[hwyV0+2] += state[hwyMul1+2]
		state[hwyMul1+2] ^= uint64(uint32(state[hwyV0+2])) * (state[hwyV1+2] >> 32)

		state[hwyV1+3] += binary.LittleEndian.Uint64(m[24:]) + state[hwyMul0+3]
		state[hwyMul0+3] ^= uint64(uint32(state[hwyV1+3])) * (state[hwyV0+3] >> 32)
		state[hwyV0+3] += state[hwyMul1+3]
		state[hwyMul1+3] ^= uint64(uint32(state[hwyV0+3])) * (state[hwyV1+3] >> 32)

		highwayZipperMerge(state[hwyV1+0], state[hwyV1+1], &state[hwyV0+0], &state[hwyV0+1])
		highwayZipperMerge(state[hwyV1+2], state[hwyV1+3], &state[hwyV0+2], &state[hwyV0+3])
		highwayZipperMerge(state[hwyV0+0], state[hwyV0+1], &state[hwyV1+0], &state[hwyV1+1])
		highwayZipperMerge(state[hwyV0+2], state[hwyV0+3], &state[hwyV1+2], &state[hwyV1+3])

		msg = msg[32:]
	}
}

func highwayZipperMerge(v0, v1 uint64, d0, d1 *uint64) {
	res := v0 & (0xff << (2 * 8))
	res2 := (v0 & (0xff << (7 * 8))) + (v1 & (0xff << (2 * 8)))
	res += (v1 & (0xff << (7 * 8))) >> 8
	res2 += (v0 & (0xff << (6 * 8))) >> 8
	res += ((v0 & (0xff << (5 * 8))) + (v1 & (0xff << (6 * 8)))) >> 16
	res2 += (v1 & (0xff << (5 * 8))) >> 16
	res += ((v0 & (0xff << (3 * 8))) + (v1 & (0xff << (4 * 8)))) >> 24
	res2 += ((v1 & (0xff << (3 * 8))) + (v0 & (0xff << (4 * 8)))) >> 24
	res += (v0 & (0xff << (1 * 8))) << 32
	res2 += (v1 & 0xff) << 48
	res += v0 << 56
	res2 += (v1 & (0xff << (1 * 8))) << 24
	*d0 += res
	*d1 += res2
}

func highwayHashBuffer(state *[16]uint64, buffer *[32]byte, offset int) {
	const blockSize = 32
	var block [blockSize]byte
	mod32 := (uint64(offset) << 32) + uint64(offset)
	for i := range state[:4] {
		state[i] += mod32
	}
	for i := range state[4:8] {
		t0 := uint32(state[i+4])
		t0 = (t0 << uint(offset)) | (t0 >> uint(32-offset))
		t1 := uint32(state[i+4] >> 32)
		t1 = (t1 << uint(offset)) | (t1 >> uint(32-offset))
		state[i+4] = (uint64(t1) << 32) | uint64(t0)
	}

	mod4 := offset & 3
	remain := offset - mod4

	copy(block[:], buffer[:remain])
	if offset >= 16 {
		copy(block[28:], buffer[offset-4:])
	} else if mod4 != 0 {
		last := uint32(buffer[remain])
		last += uint32(buffer[remain+mod4>>1]) << 8
		last += uint32(buffer[offset-1]) << 16
		binary.LittleEndian.PutUint32(block[16:], last)
	}
	highwayUpdate(state, block[:])
}

func highwayFinalize128(state *[16]uint64, out []byte) {
	var perm [4]uint64
	var tmp [32]byte
	const runs = 6
	for i := 0; i < runs; i++ {
		perm[0] = state[hwyV0+2]>>32 | state[hwyV0+2]<<32
		perm[1] = state[hwyV0+3]>>32 | state[hwyV0+3]<<32
		perm[2] = state[hwyV0+0]>>32 | state[hwyV0+0]<<32
		perm[3] = state[hwyV0+1]>>32 | state[hwyV0+1]<<32

		binary.LittleEndian.PutUint64(tmp[0:], perm[0])
		binary.LittleEndian.PutUint64(tmp[8:], perm[1])
		binary.LittleEndian.PutUint64(tmp[16:], perm[2])
		binary.LittleEndian.PutUint64(tmp[24:], perm[3])

		highwayUpdate(state, tmp[:])
	}

	binary.LittleEndian.PutUint64(out, state[hwyV0+0]+state[hwyV1+2]+state[hwyMul0+0]+state[hwyMul1+2])
	binary.LittleEndian.PutUint64(out[8:], state[hwyV0+1]+state[hwyV1+3]+state[hwyMul0+1]+state[hwyMul1+3])
}
