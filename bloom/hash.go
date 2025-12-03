package bloom

import (
	"hash/fnv"
)

// deriveHashes produces m bit positions for key using double hashing.
// h(i) = (h1 + i*h2) mod m
func deriveHashes(key []byte, m uint64, k uint) (h1, h2 uint64) {
	h := fnv.New64a()
	_, _ = h.Write(key)
	h1 = h.Sum64()

	h.Reset()
	_, _ = h.Write(key)
	_, _ = h.Write([]byte{0})
	h2 = h.Sum64()

	if h2 == 0 {
		h2 = 1
	}
	return h1, h2
}

func bitIndex(h1, h2 uint64, m uint64, i uint) uint64 {
	return (h1 + uint64(i)*h2) % m
}
