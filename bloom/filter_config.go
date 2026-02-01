package bloom

import "fmt"

// FilterOption customizes a FilterConfig after sizing is set.
type FilterOption func(*FilterConfig)

// WithFilterHash sets the hash strategy on a filter config.
func WithFilterHash(strategy Strategy) FilterOption {
	return func(fc *FilterConfig) {
		fc.Hash.Strategy = strategy
	}
}

// WithFilterSeed sets the hash seed on a filter config.
func WithFilterSeed(seed uint64) FilterOption {
	return func(fc *FilterConfig) {
		fc.Hash.Seed = seed
	}
}

// WithFilterHashConfig sets the full hash configuration on a filter config.
func WithFilterHashConfig(hash HashConfig) FilterOption {
	return func(fc *FilterConfig) {
		fc.Hash = hash
	}
}

// WithFilterSizingBounds sets target sizing bounds on a filter config.
func WithFilterSizingBounds(bounds SizingBounds) FilterOption {
	return func(fc *FilterConfig) {
		if fc.Sizing.Mode != SizingTarget {
			return
		}
		fc.Sizing.Target.Bounds = bounds
	}
}

// WithFilterRecommendedHash tunes hash strategy and seed against the resolved layout.
func WithFilterRecommendedHash(rec RecommendedHashOptions) FilterOption {
	return func(fc *FilterConfig) {
		tuned, err := fc.WithRecommendedHash(rec)
		if err != nil {
			panic(fmt.Errorf("bloom: WithFilterRecommendedHash: %w", err))
		}
		*fc = tuned
	}
}

// SizingConfig names how m and k are chosen using a discriminated mode field.
// Only the branch matching Mode is read during validation and sizing.
type SizingConfig struct {
	Mode     SizingMode
	Target   TargetSpec   // valid when Mode == SizingTarget
	Explicit ExplicitSpec // valid when Mode == SizingExplicit
}

// TargetSizing returns sizing that derives m and k from capacity and FPR.
func TargetSizing(capacity uint64, fpr float64, bounds SizingBounds) SizingConfig {
	return SizingConfig{
		Mode: SizingTarget,
		Target: TargetSpec{
			Capacity: capacity,
			FPR:      fpr,
			Bounds:   bounds,
		},
	}
}

// ExplicitSizing returns sizing with fixed bit count and hash functions.
func ExplicitSizing(bits uint64, hashCount uint) SizingConfig {
	return SizingConfig{
		Mode: SizingExplicit,
		Explicit: ExplicitSpec{
			Bits:      bits,
			HashCount: hashCount,
		},
	}
}

// Validate checks that sizing inputs for the active mode are usable.
func (s SizingConfig) Validate() error {
	switch s.Mode {
	case SizingTarget:
		return s.Target.Validate()
	case SizingExplicit:
		return s.Explicit.Validate()
	default:
		return fmt.Errorf("bloom: unknown sizing mode %v", s.Mode)
	}
}

// Size resolves m (bits) and k (hash functions) from the sizing configuration.
func (s SizingConfig) Size() (m uint64, k uint, err error) {
	if err = s.Validate(); err != nil {
		return 0, 0, err
	}

	switch s.Mode {
	case SizingExplicit:
		k = s.Explicit.HashCount
		if k == 0 {
			k = 1
		}
		return s.Explicit.Bits, k, nil
	case SizingTarget:
		b := s.Target.Bounds.Resolved()
		m = optimalM(s.Target.Capacity, s.Target.FPR, b.MinBits)
		k = optimalK(m, s.Target.Capacity, b.MaxHashCount)
		return m, k, nil
	default:
		return 0, 0, fmt.Errorf("bloom: unknown sizing mode %v", s.Mode)
	}
}

// String summarizes sizing for debugging and CLI output.
func (s SizingConfig) String() string {
	switch s.Mode {
	case SizingTarget:
		return fmt.Sprintf("target n=%d p=%g bounds=%s", s.Target.Capacity, s.Target.FPR, s.Target.Bounds)
	case SizingExplicit:
		k := s.Explicit.HashCount
		if k == 0 {
			k = 1
		}
		return fmt.Sprintf("explicit m=%d k=%d", s.Explicit.Bits, k)
	default:
		return fmt.Sprintf("invalid sizing mode %v", s.Mode)
	}
}

// FilterConfig describes how a Bloom filter is sized and hashed using separate,
// typed fields. Prefer BuildFilterConfig or the TargetFilter / ExplicitFilter
// helpers for validated construction; use Config() when calling legacy APIs.
type FilterConfig struct {
	Sizing SizingConfig
	Hash   HashConfig
}

// CountingConfig extends FilterConfig with per-bit counter width for counting filters.
type CountingConfig struct {
	Filter       FilterConfig
	CounterWidth uint8 // 8 (default), 16, 32, or 64
}

// TargetFilter returns a FilterConfig that derives m and k from capacity and FPR.
// Like TargetConfig, validation is deferred until Size or Validate is called.
func TargetFilter(capacity uint64, fpr float64, opts ...FilterOption) FilterConfig {
	return FilterFromTarget(TargetSpec{
		Capacity: capacity,
		FPR:      fpr,
	}, opts...)
}

// ExplicitFilter returns a FilterConfig with fixed bit count and hash functions.
func ExplicitFilter(bits uint64, hashCount uint, opts ...FilterOption) FilterConfig {
	return FilterFromExplicit(ExplicitSpec{
		Bits:      bits,
		HashCount: hashCount,
	}, opts...)
}

// FilterFromTarget builds a FilterConfig from typed target sizing inputs.
func FilterFromTarget(spec TargetSpec, opts ...FilterOption) FilterConfig {
	fc := FilterConfig{Sizing: TargetSizing(spec.Capacity, spec.FPR, spec.Bounds)}
	applyFilterOptions(&fc, opts)
	return fc
}

// FilterFromExplicit builds a FilterConfig from typed explicit sizing inputs.
func FilterFromExplicit(spec ExplicitSpec, opts ...FilterOption) FilterConfig {
	fc := FilterConfig{Sizing: ExplicitSizing(spec.Bits, spec.HashCount)}
	applyFilterOptions(&fc, opts)
	return fc
}

// BuildFilterConfig constructs and validates a FilterConfig from typed sizing specs.
func BuildFilterConfig(mode SizingMode, target TargetSpec, explicit ExplicitSpec, opts ...FilterOption) (FilterConfig, error) {
	var fc FilterConfig
	switch mode {
	case SizingTarget:
		if err := target.Validate(); err != nil {
			return FilterConfig{}, err
		}
		fc = FilterFromTarget(target, opts...)
	case SizingExplicit:
		if err := explicit.Validate(); err != nil {
			return FilterConfig{}, err
		}
		fc = FilterFromExplicit(explicit, opts...)
	default:
		return FilterConfig{}, fmt.Errorf("bloom: unknown sizing mode %v", mode)
	}
	if err := fc.Validate(); err != nil {
		return FilterConfig{}, err
	}
	return fc, nil
}

// TargetCounting returns a CountingConfig sized from capacity and FPR.
func TargetCounting(capacity uint64, fpr float64, opts ...CountingOption) CountingConfig {
	cc := CountingConfig{Filter: TargetFilter(capacity, fpr)}
	applyCountingOptions(&cc, opts)
	return cc
}

// ExplicitCounting returns a CountingConfig with fixed m and k.
func ExplicitCounting(bits uint64, hashCount uint, opts ...CountingOption) CountingConfig {
	cc := CountingConfig{Filter: ExplicitFilter(bits, hashCount)}
	applyCountingOptions(&cc, opts)
	return cc
}

// BuildCountingConfig validates sizing and counter width, returning a CountingConfig.
func BuildCountingConfig(mode SizingMode, target TargetSpec, explicit ExplicitSpec, opts ...CountingOption) (CountingConfig, error) {
	fc, err := BuildFilterConfig(mode, target, explicit)
	if err != nil {
		return CountingConfig{}, err
	}
	cc := CountingConfig{Filter: fc}
	applyCountingOptions(&cc, opts)
	if err := cc.Validate(); err != nil {
		return CountingConfig{}, err
	}
	return cc, nil
}

// CountingOption customizes a CountingConfig.
type CountingOption func(*CountingConfig)

// WithCountingHash sets the hash strategy on a counting config.
func WithCountingHash(strategy Strategy) CountingOption {
	return func(cc *CountingConfig) {
		cc.Filter.Hash.Strategy = strategy
	}
}

// WithCountingSeed sets the hash seed on a counting config.
func WithCountingSeed(seed uint64) CountingOption {
	return func(cc *CountingConfig) {
		cc.Filter.Hash.Seed = seed
	}
}

// WithCountingHashConfig sets the full hash configuration on a counting config.
func WithCountingHashConfig(hash HashConfig) CountingOption {
	return func(cc *CountingConfig) {
		cc.Filter.Hash = hash
	}
}

// WithCountingSizingBounds sets target sizing bounds on a counting config.
func WithCountingSizingBounds(bounds SizingBounds) CountingOption {
	return func(cc *CountingConfig) {
		WithFilterSizingBounds(bounds)(&cc.Filter)
	}
}

// WithCountingCounterWidth selects per-bit counter width (8, 16, 32, or 64).
func WithCountingCounterWidth(width uint8) CountingOption {
	return func(cc *CountingConfig) {
		cc.CounterWidth = width
	}
}

// Target returns target sizing inputs when Mode is SizingTarget.
func (fc FilterConfig) Target() (TargetSpec, bool) {
	if fc.Sizing.Mode != SizingTarget {
		return TargetSpec{}, false
	}
	return fc.Sizing.Target, true
}

// Explicit returns explicit sizing inputs when Mode is SizingExplicit.
func (fc FilterConfig) Explicit() (ExplicitSpec, bool) {
	if fc.Sizing.Mode != SizingExplicit {
		return ExplicitSpec{}, false
	}
	return fc.Sizing.Explicit, true
}

// ExpectedCapacity returns the benchmark item count for target sizing, or m for explicit sizing.
func (fc FilterConfig) ExpectedCapacity() uint64 {
	if spec, ok := fc.Target(); ok {
		return spec.Capacity
	}
	return fc.Sizing.Explicit.Bits
}

// FalsePositiveRate returns the target FPR in target mode, or zero in explicit mode.
func (fc FilterConfig) FalsePositiveRate() float64 {
	if spec, ok := fc.Target(); ok {
		return spec.FPR
	}
	return 0
}

// WithFalsePositiveRate returns a copy with an updated target FPR.
func (fc FilterConfig) WithFalsePositiveRate(p float64) FilterConfig {
	out := fc
	if out.Sizing.Mode == SizingTarget {
		out.Sizing.Target.FPR = p
	}
	return out
}

// WithExpectedCapacity returns a copy with an updated target capacity.
func (fc FilterConfig) WithExpectedCapacity(n uint64) FilterConfig {
	out := fc
	if out.Sizing.Mode == SizingTarget {
		out.Sizing.Target.Capacity = n
	}
	return out
}

// WithHashStrategy returns a copy using the given hash strategy.
func (fc FilterConfig) WithHashStrategy(strategy Strategy) FilterConfig {
	out := fc
	out.Hash.Strategy = strategy
	return out
}

// Apply returns a copy of fc with the given options applied.
func (fc FilterConfig) Apply(opts ...FilterOption) FilterConfig {
	applyFilterOptions(&fc, opts)
	return fc
}

// Validate checks that sizing and hash settings are usable.
func (fc FilterConfig) Validate() error {
	return fc.Sizing.Validate()
}

// Size resolves m and k from the filter configuration.
func (fc FilterConfig) Size() (m uint64, k uint, err error) {
	return fc.Sizing.Size()
}

// Mode reports whether target or explicit sizing is active.
func (fc FilterConfig) Mode() SizingMode {
	return fc.Sizing.Mode
}

// Hasher returns the configured hash implementation.
func (fc FilterConfig) Hasher() Hasher {
	return fc.Hash.Hasher()
}

// String summarizes the resolved filter configuration.
func (fc FilterConfig) String() string {
	m, k, err := fc.Size()
	if err != nil {
		return fmt.Sprintf("invalid filter config: %v", err)
	}
	hash := fc.Hash.String()
	switch fc.Sizing.Mode {
	case SizingExplicit:
		return fmt.Sprintf("explicit m=%d k=%d hash=%s", m, k, hash)
	case SizingTarget:
		return fmt.Sprintf("target n=%d p=%g -> m=%d k=%d hash=%s",
			fc.Sizing.Target.Capacity, fc.Sizing.Target.FPR, m, k, hash)
	default:
		return fmt.Sprintf("invalid filter config: unknown mode %v", fc.Sizing.Mode)
	}
}

// WithRecommendedHash returns a copy whose hash settings were chosen by RecommendHasher.
func (fc FilterConfig) WithRecommendedHash(rec RecommendedHashOptions) (FilterConfig, error) {
	cfg, err := fc.Config().WithRecommendedHash(rec)
	if err != nil {
		return FilterConfig{}, err
	}
	return cfg.FilterConfig(), nil
}

// Config flattens fc into the legacy Config representation.
func (fc FilterConfig) Config() Config {
	switch fc.Sizing.Mode {
	case SizingTarget:
		return ConfigFromTarget(fc.Sizing.Target,
			WithHashConfig(fc.Hash),
		)
	case SizingExplicit:
		return ConfigFromExplicit(fc.Sizing.Explicit,
			WithHashConfig(fc.Hash),
		)
	default:
		return Config{Hash: fc.Hash}
	}
}

// FilterConfig converts a legacy Config into the structured FilterConfig form.
func (c Config) FilterConfig() FilterConfig {
	switch c.Mode() {
	case SizingExplicit:
		spec, _ := c.Explicit()
		return FilterConfig{
			Sizing: ExplicitSizing(spec.Bits, spec.HashCount),
			Hash:   c.Hash,
		}
	default:
		spec, _ := c.Target()
		return FilterConfig{
			Sizing: TargetSizing(spec.Capacity, spec.FPR, spec.Bounds),
			Hash:   c.Hash,
		}
	}
}

// Validate checks sizing and counter width for counting filters.
func (cc CountingConfig) Validate() error {
	if err := cc.Filter.Validate(); err != nil {
		return err
	}
	return cc.validateCounterWidth()
}

func (cc CountingConfig) resolvedCounterWidth() uint8 {
	if cc.CounterWidth == 0 || cc.CounterWidth == 8 {
		return 8
	}
	return cc.CounterWidth
}

func (cc CountingConfig) validateCounterWidth() error {
	switch cc.resolvedCounterWidth() {
	case 8, 16, 32, 64:
		return nil
	default:
		return ErrInvalidCounterWidth
	}
}

// Config flattens cc into the legacy Config representation.
func (cc CountingConfig) Config() Config {
	cfg := cc.Filter.Config()
	cfg.CounterWidth = cc.CounterWidth
	return cfg
}

// CountingConfig converts a legacy Config into structured counting configuration.
func (c Config) CountingConfig() CountingConfig {
	return CountingConfig{
		Filter:       c.FilterConfig(),
		CounterWidth: c.CounterWidth,
	}
}

func applyFilterOptions(fc *FilterConfig, opts []FilterOption) {
	for _, opt := range opts {
		opt(fc)
	}
}

func applyCountingOptions(cc *CountingConfig, opts []CountingOption) {
	for _, opt := range opts {
		opt(cc)
	}
}
