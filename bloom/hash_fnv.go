package bloom

import "hash/fnv"

type fnvHasher struct{}

func (fnvHasher) Strategy() Strategy { return HashFNV }

// Derive matches the original bloomlab scheme: FNV-1a(key) and FNV-1a(key‖0x00).
func (fnvHasher) Derive(key []byte) (h1, h2 uint64) {
	h := fnv.New64a()
	_, _ = h.Write(key)
	h1 = h.Sum64()

	h.Reset()
	_, _ = h.Write(key)
	var tail [1]byte
	_, _ = h.Write(tail[:])
	h2 = ensureH2NonZero(h.Sum64())
	return h1, h2
}
