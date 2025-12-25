package bloom

import (
	"errors"
	"fmt"
	"math"
)

var (
	ErrInvalidCapacity    = errors.New("bloom: capacity must be positive")
	ErrInvalidFPR         = errors.New("bloom: false positive rate must be in (0, 1)")
	ErrInvalidBits        = errors.New("bloom: bit count must be positive")
	ErrInvalidCounterWidth = errors.New("bloom: counter width must be 8 or 16")
)

const (
	defaultMinBits      uint64 = 64
	defaultMaxHashCount uint   = 32
)

// HashConfig selects how keys are hashed into bit positions. Sizing and hashing
// are independent: hash settings do not affect m or k.
type HashConfig struct {
	// Strategy selects the hash family for double hashing. Zero means HashFNV.
	Strategy Strategy
	// Seed seeds keyed hashing; use distinct seeds for independent filters.
	Seed uint64
}

// Hasher returns the configured hash implementation.
func (h HashConfig) Hasher() Hasher {
	return NewHasher(h.Strategy, h.Seed)
}

// String returns a short hash summary for debugging and CLI output.
func (h HashConfig) String() string {
	name := h.Strategy.String()
	if h.Seed != 0 {
		return fmt.Sprintf("%s seed=%d", name, h.Seed)
	}
	return name
}

// ConfigOption customizes a Config after sizing fields are set.
type ConfigOption func(*Config)

// WithHash sets the hash strategy on a config.
func WithHash(strategy Strategy) ConfigOption {
	return func(c *Config) {
		c.Hash.Strategy = strategy
	}
}

// WithSeed sets the hash seed on a config.
func WithSeed(seed uint64) ConfigOption {
	return func(c *Config) {
		c.Hash.Seed = seed
	}
}

// WithHashConfig sets the full hash configuration on a config.
func WithHashConfig(hash HashConfig) ConfigOption {
	return func(c *Config) {
		c.Hash = hash
	}
}

// WithMinBits sets the minimum bit count for target-based sizing.
func WithMinBits(minBits uint64) ConfigOption {
	return func(c *Config) {
		c.MinBits = minBits
	}
}

// WithMaxHashCount caps the number of hash functions for target-based sizing.
func WithMaxHashCount(maxK uint) ConfigOption {
	return func(c *Config) {
		c.MaxHashCount = maxK
	}
}

// WithCounterWidth selects per-bit counter width for counting filters.
// Supported values are 8 (default) and 16. Wider counters use more memory
// but tolerate more duplicate inserts before ErrCounterOverflow.
func WithCounterWidth(width uint8) ConfigOption {
	return func(c *Config) {
		c.CounterWidth = width
	}
}

// Config describes how a Bloom filter is sized. Use TargetConfig for the
// standard capacity/FPR formulas, or ExplicitConfig for fixed m and k.
// Hash settings live in Config.Hash and can be supplied via WithHash, WithSeed,
// or WithHashConfig. Sizing bounds use WithMinBits and WithMaxHashCount.
type Config struct {
	ExpectedCapacity  uint64
	FalsePositiveRate float64

	// Bits (m) and HashCount (k) select explicit sizing when Bits is non-zero.
	Bits      uint64
	HashCount uint

	// MinBits and MaxHashCount bound derived sizing. Zero values use package defaults.
	MinBits      uint64
	MaxHashCount uint

	// CounterWidth selects uint8 (0 or 8) or uint16 (16) counters for counting filters.
	CounterWidth uint8

	Hash HashConfig
}

// TargetConfig returns a Config that derives m and k from expected capacity
// and false positive rate using the standard Bloom filter formulas.
func TargetConfig(expectedCapacity uint64, falsePositiveRate float64, opts ...ConfigOption) Config {
	cfg := Config{
		ExpectedCapacity:  expectedCapacity,
		FalsePositiveRate: falsePositiveRate,
	}
	applyOptions(&cfg, opts)
	return cfg
}

// ExplicitConfig returns a Config with fixed bit count and hash functions.
// HashCount of zero is treated as one at construction time.
func ExplicitConfig(bits uint64, hashCount uint, opts ...ConfigOption) Config {
	cfg := Config{
		Bits:      bits,
		HashCount: hashCount,
	}
	applyOptions(&cfg, opts)
	return cfg
}

// Validate checks that the configuration is usable.
func (c Config) Validate() error {
	if c.Bits != 0 {
		return nil
	}
	if c.HashCount > 0 && c.ExpectedCapacity == 0 && c.FalsePositiveRate == 0 {
		return ErrInvalidBits
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
	return c.Hash.Hasher()
}

// String summarizes the resolved sizing for debugging and CLI output.
func (c Config) String() string {
	m, k, err := c.Size()
	if err != nil {
		return fmt.Sprintf("invalid config: %v", err)
	}
	hash := c.Hash.String()
	if c.Bits != 0 {
		return fmt.Sprintf("explicit m=%d k=%d hash=%s", m, k, hash)
	}
	return fmt.Sprintf("target n=%d p=%g -> m=%d k=%d hash=%s", c.ExpectedCapacity, c.FalsePositiveRate, m, k, hash)
}

func applyOptions(cfg *Config, opts []ConfigOption) {
	for _, opt := range opts {
		opt(cfg)
	}
}

func (c Config) resolvedCounterWidth() uint8 {
	if c.CounterWidth == 0 || c.CounterWidth == 8 {
		return 8
	}
	return c.CounterWidth
}

func (c Config) validateCounterWidth() error {
	switch c.resolvedCounterWidth() {
	case 8, 16:
		return nil
	default:
		return ErrInvalidCounterWidth
	}
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
