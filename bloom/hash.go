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
	// HashWyhash uses wyhash final v1 64-bit with independent seeds for h1 and h2.
	HashWyhash
	// HashHighway uses HighwayHash-128 in a single pass for h1 and h2 (seed-sensitive).
	HashHighway
)

// AllStrategies returns every built-in hash strategy in stable order.
func AllStrategies() []Strategy {
	return []Strategy{HashFNV, HashMurmur3, HashXXHash, HashWyhash, HashHighway}
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
	case HashWyhash:
		return "wyhash"
	case HashHighway:
		return "highway"
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
	case "wyhash", "wy":
		return HashWyhash, nil
	case "highway", "highwayhash", "hwy":
		return HashHighway, nil
	default:
		return 0, fmt.Errorf("bloom: unknown hash strategy %q (want fnv, murmur3, xxhash, wyhash, or highway)", name)
	}
}

// ParseStrategyList parses comma-separated hash strategy names.
func ParseStrategyList(raw string) ([]Strategy, error) {
	parts := strings.Split(raw, ",")
	strategies := make([]Strategy, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		s, err := ParseStrategy(part)
		if err != nil {
			return nil, err
		}
		strategies = append(strategies, s)
	}
	if len(strategies) == 0 {
		return nil, fmt.Errorf("bloom: no hash strategies in %q", raw)
	}
	return strategies, nil
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
	case HashWyhash:
		return wyhashHasher{seed: seed}
	case HashHighway:
		return highwayHasher{key: expandSeedToHighwayKey(seed)}
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

// doubleHashSeedMix separates the two independent hash passes used in double hashing.
// It is the 64-bit golden-ratio constant also used by SplitMix64-style seed expansion.
const doubleHashSeedMix = uint64(0x9e3779b97f4a7c15)

// pairDerivedSeeds returns independent seeds for h1 and h2 from a filter seed.
func pairDerivedSeeds(seed uint64) (seed1, seed2 uint64) {
	return seed, seed ^ doubleHashSeedMix
}

// ensureH2NonZero guarantees h2 is usable for double hashing (h2 must not be 0).
func ensureH2NonZero(h2 uint64) uint64 {
	if h2 == 0 {
		return 1
	}
	return h2
}
