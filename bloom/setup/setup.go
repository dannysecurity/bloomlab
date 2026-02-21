// Package setup provides a fluent builder for Bloom filter configuration.
// It unifies target and explicit sizing, hash tuning, and counting options
// behind one entry point so callers do not split options across FilterOption
// and CountingOption helpers.
package setup

import (
	"fmt"

	"github.com/dannysecurity/bloomlab/bloom"
)

// Option customizes a Builder before FilterConfig or CountingConfig is built.
type Option func(*Builder)

// WithHash sets the hash strategy.
func WithHash(strategy bloom.Strategy) Option {
	return func(b *Builder) {
		b.hash.Strategy = strategy
	}
}

// WithSeed sets the hash seed.
func WithSeed(seed uint64) Option {
	return func(b *Builder) {
		b.hash.Seed = seed
	}
}

// WithHashConfig sets the full hash configuration.
func WithHashConfig(hash bloom.HashConfig) Option {
	return func(b *Builder) {
		b.hash = hash
	}
}

// WithSizingBounds sets target sizing bounds. Has no effect in explicit mode.
func WithSizingBounds(bounds bloom.SizingBounds) Option {
	return func(b *Builder) {
		if b.sizing.Mode != bloom.SizingTarget {
			return
		}
		b.sizing.Target.Bounds = bounds
	}
}

// WithCounterWidth selects per-bit counter width for counting filters (2, 4, 8, 16, 32, or 64).
func WithCounterWidth(width uint8) Option {
	return func(b *Builder) {
		b.counterWidth = width
	}
}

// WithRecommendedHash tunes hash strategy and seed against the resolved layout.
func WithRecommendedHash(rec bloom.RecommendedHashOptions) Option {
	return func(b *Builder) {
		fc, err := b.filterConfig()
		if err != nil {
			panic(fmt.Errorf("setup: WithRecommendedHash: %w", err))
		}
		tuned, err := fc.WithRecommendedHash(rec)
		if err != nil {
			panic(fmt.Errorf("setup: WithRecommendedHash: %w", err))
		}
		b.sizing = tuned.Sizing
		b.hash = tuned.Hash
	}
}

// Builder assembles filter or counting configuration from sizing, hash, and
// optional counter width. Construct with Target, Explicit, or FromSizing.
type Builder struct {
	sizing       bloom.SizingConfig
	hash         bloom.HashConfig
	counterWidth uint8
}

// Target returns a builder that derives m and k from capacity and FPR.
func Target(capacity uint64, fpr float64, opts ...Option) *Builder {
	return FromSizing(bloom.TargetSizing(capacity, fpr, bloom.SizingBounds{}), opts...)
}

// Explicit returns a builder with fixed bit count and hash functions.
func Explicit(bits uint64, hashCount uint, opts ...Option) *Builder {
	return FromSizing(bloom.ExplicitSizing(bits, hashCount), opts...)
}

// FromSizing returns a builder from an existing SizingConfig.
func FromSizing(sizing bloom.SizingConfig, opts ...Option) *Builder {
	b := &Builder{sizing: sizing}
	b.apply(opts)
	return b
}

// Apply returns a copy with the given options applied.
func (b Builder) Apply(opts ...Option) *Builder {
	out := b
	out.apply(opts)
	return &out
}

// FilterConfig validates sizing and returns structured filter configuration.
func (b *Builder) FilterConfig() (bloom.FilterConfig, error) {
	return b.filterConfig()
}

// CountingConfig validates sizing and counter width, returning counting configuration.
func (b *Builder) CountingConfig() (bloom.CountingConfig, error) {
	fc, err := b.filterConfig()
	if err != nil {
		return bloom.CountingConfig{}, err
	}
	cc := bloom.CountingConfig{Filter: fc, CounterWidth: b.counterWidth}
	if err := cc.Validate(); err != nil {
		return bloom.CountingConfig{}, err
	}
	return cc, nil
}

// Filter constructs a standard Bloom filter from the builder state.
func (b *Builder) Filter() (*bloom.Filter, error) {
	fc, err := b.filterConfig()
	if err != nil {
		return nil, err
	}
	return bloom.NewFilterFrom(fc)
}

// CountingFilter constructs a counting Bloom filter from the builder state.
func (b *Builder) CountingFilter() (*bloom.CountingFilter, error) {
	cc, err := b.CountingConfig()
	if err != nil {
		return nil, err
	}
	return bloom.NewCountingFilterFrom(cc)
}

// Plan resolves target sizing and returns theoretical fill and FPR at capacity.
// Explicit builders return bloom.ErrInvalidCapacity.
func (b *Builder) Plan() (bloom.SizingPlan, error) {
	fc, err := b.filterConfig()
	if err != nil {
		return bloom.SizingPlan{}, err
	}
	return bloom.PlanSizingFromFilter(fc)
}

func (b *Builder) filterConfig() (bloom.FilterConfig, error) {
	if err := b.sizing.Validate(); err != nil {
		return bloom.FilterConfig{}, err
	}
	fc := bloom.FilterConfig{Sizing: b.sizing, Hash: b.hash}
	if err := fc.Validate(); err != nil {
		return bloom.FilterConfig{}, err
	}
	return fc, nil
}

func (b *Builder) apply(opts []Option) {
	for _, opt := range opts {
		opt(b)
	}
}
