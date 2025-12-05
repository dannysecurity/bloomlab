package bloom

import (
	"errors"
	"fmt"
	"math"
)

var (
	ErrInvalidCapacity = errors.New("bloom: capacity must be positive")
	ErrInvalidFPR      = errors.New("bloom: false positive rate must be in (0, 1)")
)

const (
	defaultMinBits      uint64 = 64
	defaultMaxHashCount uint   = 32
)

// Config describes how a Bloom filter is sized. Use TargetConfig for the
// standard capacity/FPR formulas, or ExplicitConfig for fixed m and k.
type Config struct {
	ExpectedCapacity  uint64
	FalsePositiveRate float64

	// Bits (m) and HashCount (k) select explicit sizing when Bits is non-zero.
	Bits      uint64
	HashCount uint

	// MinBits and MaxHashCount bound derived sizing. Zero values use package defaults.
	MinBits      uint64
	MaxHashCount uint

	// HashStrategy selects the hash family for double hashing. Zero means HashFNV.
	HashStrategy Strategy
	// HashSeed seeds keyed hashing; use distinct seeds for independent filters.
	HashSeed uint64
}

// TargetConfig returns a Config that derives m and k from expected capacity
// and false positive rate using the standard Bloom filter formulas.
func TargetConfig(expectedCapacity uint64, falsePositiveRate float64) Config {
	return Config{
		ExpectedCapacity:  expectedCapacity,
		FalsePositiveRate: falsePositiveRate,
	}
}

// ExplicitConfig returns a Config with fixed bit count and hash functions.
// HashCount of zero is treated as one at construction time.
func ExplicitConfig(bits uint64, hashCount uint) Config {
	return Config{
		Bits:      bits,
		HashCount: hashCount,
	}
}

// Validate checks that the configuration is usable.
func (c Config) Validate() error {
	if c.Bits != 0 {
		return nil
	}
	if c.ExpectedCapacity == 0 {
		return ErrInvalidCapacity
	}
	if c.FalsePositiveRate <= 0 || c.FalsePositiveRate >= 1 {
		return ErrInvalidFPR
	}
	return nil
}

// Size resolves m (bits) and k (hash functions) from the configuration.
func (c Config) Size() (m uint64, k uint, err error) {
	if err = c.Validate(); err != nil {
		return 0, 0, err
	}

	if c.Bits != 0 {
		k = c.HashCount
		if k == 0 {
			k = 1
		}
		return c.Bits, k, nil
	}

	minBits, maxK := c.bounds()
	m = optimalM(c.ExpectedCapacity, c.FalsePositiveRate, minBits)
	k = optimalK(m, c.ExpectedCapacity, maxK)
	return m, k, nil
}

// Hasher returns the configured hash implementation.
func (c Config) Hasher() Hasher {
	return NewHasher(c.HashStrategy, c.HashSeed)
}

// String summarizes the resolved sizing for debugging and CLI output.
func (c Config) String() string {
	m, k, err := c.Size()
	if err != nil {
		return fmt.Sprintf("invalid config: %v", err)
	}
	hash := c.HashStrategy.String()
	if c.HashSeed != 0 {
		hash = fmt.Sprintf("%s seed=%d", hash, c.HashSeed)
	}
	if c.Bits != 0 {
		return fmt.Sprintf("explicit m=%d k=%d hash=%s", m, k, hash)
	}
	return fmt.Sprintf("target n=%d p=%g -> m=%d k=%d hash=%s", c.ExpectedCapacity, c.FalsePositiveRate, m, k, hash)
}

func (c Config) bounds() (minBits uint64, maxK uint) {
	minBits = c.MinBits
	if minBits == 0 {
		minBits = defaultMinBits
	}
	maxK = c.MaxHashCount
	if maxK == 0 {
		maxK = defaultMaxHashCount
	}
	return minBits, maxK
}

func optimalM(n uint64, p float64, minBits uint64) uint64 {
	// m = -n * ln(p) / (ln(2)^2)
	m := -float64(n) * math.Log(p) / (math.Ln2 * math.Ln2)
	if m < float64(minBits) {
		return minBits
	}
	return uint64(m)
}

func optimalK(m uint64, n uint64, maxK uint) uint {
	// k = (m/n) * ln(2)
	k := float64(m) / float64(n) * math.Ln2
	if k < 1 {
		return 1
	}
	if k > float64(maxK) {
		return maxK
	}
	return uint(k)
}
