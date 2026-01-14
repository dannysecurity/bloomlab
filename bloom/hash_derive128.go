package bloom

// splitDoubleHashPair returns (h1, h2) from a single 128-bit digest for double hashing.
// h2 is forced non-zero so bitIndex(h1, h2, m, i) can advance with i*h2.
func splitDoubleHashPair(lo, hi uint64) (h1, h2 uint64) {
	return lo, ensureH2NonZero(hi)
}
