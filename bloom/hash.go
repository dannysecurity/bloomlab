package bloom

import (
	"fmt"
	"strings"
)

// Strategy selects the hash family used for double hashing.
type Strategy int

const (
	// HashFNV uses FNV-1a 64-bit with a second pass over key‖0x00 (default).
	HashFNV Strategy = iota
	// HashMurmur3 uses MurmurHash3 64-bit with independent seeds for h1 and h2.
	HashMurmur3
	// HashXXHash uses xxHash 64-bit with independent seeds for h1 and h2.
	HashXXHash
)

// AllStrategies returns every built-in hash strategy in stable order.
func AllStrategies() []Strategy {
	return []Strategy{HashFNV, HashMurmur3, HashXXHash}
}

// String returns the CLI-friendly strategy name.
func (s Strategy) String() string {
	switch s {
	case HashFNV:
		return "fnv"
	case HashMurmur3:
		return "murmur3"
	case HashXXHash:
		return "xxhash"
	default:
		return fmt.Sprintf("strategy(%d)", int(s))
	}
}

// ParseStrategy maps a name to a Strategy. Names are case-insensitive.
func ParseStrategy(name string) (Strategy, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", "fnv":
		return HashFNV, nil
	case "murmur3", "murmur":
		return HashMurmur3, nil
	case "xxhash", "xxh64", "xxh":
		return HashXXHash, nil
	default:
		return 0, fmt.Errorf("bloom: unknown hash strategy %q (want fnv, murmur3, or xxhash)", name)
	}
}

// Hasher derives the two 64-bit seeds used for double hashing:
// bit index i is (h1 + i*h2) mod m.
type Hasher interface {
	Strategy() Strategy
	Derive(key []byte) (h1, h2 uint64)
}

// NewHasher constructs a Hasher for the given strategy and seed.
// Seed lets independent filters share sizing but produce uncorrelated bit patterns.
func NewHasher(strategy Strategy, seed uint64) Hasher {
	switch strategy {
	case HashMurmur3:
		return murmur3Hasher{seed: seed}
	case HashXXHash:
		return xxhashHasher{seed: seed}
	default:
		return fnvHasher{}
	}
}

// bitIndex maps double-hash iteration i into a bit offset in [0, m).
func bitIndex(h1, h2 uint64, m uint64, i uint) uint64 {
	sum := h1 + uint64(i)*h2
	if m > 0 && m&(m-1) == 0 {
		return sum & (m - 1)
	}
	return sum % m
}

// ensureH2NonZero guarantees h2 is usable for double hashing (h2 must not be 0).
func ensureH2NonZero(h2 uint64) uint64 {
	if h2 == 0 {
		return 1
	}
	return h2
}
