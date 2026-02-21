# bloomlab

A Go toolkit for [Bloom filters](https://en.wikipedia.org/wiki/Bloom_filter): space-efficient probabilistic membership sets with configurable false-positive rates.

## Features

- **Standard Bloom filter** (`bloom.Filter`) — fixed-size bit array, optimal `m` and `k` from target capacity and FPR
- **Counting Bloom filter** (`bloom.CountingFilter`) — supports deletion via per-bit counters; optional 2-bit or 4-bit packed storage reduces counter memory when duplicate counts stay small
- **Benchmarks** — add, lookup, and remove throughput
- **benchcompare** — programmatic Bloom filter vs `map[string]struct{}` comparison with CLI and Go benchmarks
- **dedup** — check-then-insert stream dedup over standard or counting Bloom filters
- **Demo CLIs** — `bloomdemo`, `countingdemo`, `dedupdemo`, `streamdedup`, `countingdedup`, `urldedup`, `countingurldedup`, `fprcalc`, `hashtune`, and `benchcompare` for interactive exploration

## Install

```bash
go get github.com/dannysecurity/bloomlab
```

## Quick start

Configuration is expressed through `bloom.FilterConfig`, which separates sizing from hashing. Hash settings do not affect `m` or `k`:

```go
package main

import (
	"fmt"
	"github.com/dannysecurity/bloomlab/bloom"
)

func main() {
	fc := bloom.TargetFilter(10_000, 0.01) // ~10k items, ~1% FPR
	f, _ := bloom.NewFilterFrom(fc)
	f.Add([]byte("user:42"))
	fmt.Println(f.Contains([]byte("user:42"))) // true
	fmt.Println(fc.String())                   // target n=10000 p=0.01 -> m=... k=...
}
```

Pass hash settings as options instead of mutating the config after construction:

```go
fc := bloom.TargetFilter(10_000, 0.01,
	bloom.WithFilterHash(bloom.HashMurmur3),
	bloom.WithFilterSeed(42),
)
f, _ := bloom.NewFilterFrom(fc)
```

For a single fluent entry point that covers sizing, hash, and counting options, use the `setup` package:

```go
import "github.com/dannysecurity/bloomlab/bloom/setup"

f, _ := setup.Target(10_000, 0.01,
	setup.WithHash(bloom.HashMurmur3),
	setup.WithSeed(42),
).Filter()

cf, _ := setup.Target(10_000, 0.01,
	setup.WithCounterWidth(16),
).CountingFilter()

plan, _ := setup.Target(10_000, 0.01).Plan() // theoretical m, k, and FPR
```

`setup.Builder` validates once and can produce `FilterConfig`, `CountingConfig`, or constructed filters directly. CLI tools in this repo parse flags through the same builder path via `filterflags`.

The legacy `bloom.Config` type remains available for older call sites:

```go
cfg := bloom.TargetConfig(10_000, 0.01)
f, _ := bloom.NewFilter(cfg)
```

Shorthand constructors remain available:

```go
f, _ := bloom.New(10_000, 0.01)
```

### Counting variant (with deletion)

```go
fc := bloom.TargetFilter(10_000, 0.01)
cf, _ := bloom.NewCountingFilterFrom(bloom.CountingConfig{Filter: fc})
_ = cf.Add([]byte("session:abc"))
cf.Remove([]byte("session:abc"))
fmt.Println(cf.Contains([]byte("session:abc"))) // false
fmt.Println(cf.TheoryFPR())                     // theoretical FPR at current insert count
```

For fixed sizing, use explicit configuration:

```go
cf, _ := bloom.NewCountingFilterFrom(bloom.ExplicitCounting(1024, 4))
```

Use 2-bit packed counters when per-position duplicate counts stay at or below 3 — counter storage uses a quarter byte per bit position:

```go
cf, _ := bloom.NewCountingFilterFrom(bloom.ExplicitCounting(1024, 4, bloom.WithCountingCounterWidth(2)))
fmt.Println(cf.CounterBytes()) // (m+3)/4 bytes instead of m
```

Use 4-bit packed counters when per-position duplicate counts stay at or below 15 — counter storage uses half a byte per bit position instead of one byte:

```go
cf, _ := bloom.NewCountingFilterFrom(bloom.ExplicitCounting(1024, 4, bloom.WithCountingCounterWidth(4)))
fmt.Println(cf.CounterBytes()) // (m+1)/2 bytes instead of m
```

Use 16-bit counters when duplicate inserts may exceed 255 per probed position:

```go
cf, _ := bloom.NewCountingFilterFrom(bloom.ExplicitCounting(1024, 4, bloom.WithCountingCounterWidth(16)))
```

Use 32-bit counters for workloads with very high per-position reference counts:

```go
cf, _ := bloom.NewCountingFilterFrom(bloom.ExplicitCounting(1024, 4, bloom.WithCountingCounterWidth(32)))
```

Use 64-bit counters when per-position counts may exceed 32-bit limits (e.g. long-lived reference-counted sets):

```go
cf, _ := bloom.NewCountingFilterFrom(bloom.ExplicitCounting(1024, 4, bloom.WithCountingCounterWidth(64)))
```

## False positive rate

A Bloom filter may report **maybe present** for keys that were never inserted. That **false positive rate (FPR)** is the probability `TargetConfig(n, p)` sizes for.

### Definition

After inserting `n` distinct keys, query a key known to be absent. Each of the `k` probed bit positions is set with probability roughly `1 - e^(-kn/m)`, treating positions as independent. All `k` must be set for a false positive:

```
p ≈ (1 - e^(-kn/m))^k
```

Equivalently, let `f ≈ 1 - e^(-kn/m)` be the **fill fraction** (expected share of bits set to 1). Then `p ≈ f^k`: a miss must land on `k` already-set bits.

There are no false negatives: if a key was inserted, `Contains` always returns true.

### Formula reference

| Symbol | Meaning |
|--------|---------|
| `n` | Distinct keys inserted |
| `m` | Bit array length |
| `k` | Hash functions (probes per key) |
| `f` | Fill fraction — expected share of bits set to 1 |
| `p` | False positive rate on absent keys |

| Formula | Role |
|---------|------|
| `f ≈ 1 - e^(-kn/m)` | Fill after `n` distinct inserts (`TheoryFillFraction`) |
| `p ≈ f^k = (1 - e^(-kn/m))^k` | FPR at insert count `n` (`TheoryFalsePositiveRate`) |
| `m = -n·ln(p) / (ln 2)²` | Optimal bits for target capacity and FPR |
| `k ≈ round((m/n)·ln 2)` | Hash count at the continuous optimum |
| `m/n ≈ -ln(p) / (ln 2)²` | Bits per item — depends on `p` only, not `n` |

At the continuous optimum, `kn/m = ln 2` so `f ≈ 1/2` and `p ≈ (1/2)^k`. For `p = 0.01`, the space formula gives `m/n ≈ 9.59` bits/item.

### Derivation (step by step)

1. **One bit stays clear.** Each insert touches `k` of `m` bits. Under independence, the probability a given bit is *not* set by one insert is `(1 - 1/m)^k ≈ e^(-k/m)`.
2. **After `n` inserts.** Repeating `n` times: `P(clear) ≈ e^(-kn/m)`.
3. **Fill fraction.** `f = P(set) ≈ 1 - e^(-kn/m)`. bloomlab exposes this as `TheoryFillFraction(n, m, k)`.
4. **False positive on a miss.** A query probes `k` positions; all must be set: `p ≈ f^k = (1 - e^(-kn/m))^k`. This is `TheoryFalsePositiveRate(n, m, k)`.

At optimal sizing, `f ≈ 1/2` and `p ≈ (1/2)^k`.

### Sizing formulas

Given target capacity `n` and desired FPR upper bound `p`, bloomlab picks `m` (bits) and `k` (hash functions) to minimize space while meeting the budget:

- `m = -n · ln(p) / (ln 2)²`
- `k = (m/n) · ln 2` (rounded, with a minimum of 1)

This `(m, k)` pair is optimal for fixed `n` and `m`: the achieved rate is close to `(1/2)^k`. The formulas appear in `optimalM` / `optimalK` inside `bloom/config.go`; `ContinuousOptimalM` / `ContinuousOptimalK` in `bloom/fpr_derivation.go` expose the real-valued values before truncation.

#### Inverse sizing (deriving `m` from `p`)

At the continuous optimum, set `k* = (m/n) · ln 2`. Then `kn/m = ln 2`, so the fill fraction is `f = 1 - e^(-ln 2) = 1/2` and the false-positive rate becomes:

```
p ≈ f^k* = (1/2)^k* = 2^(-(m/n)·ln 2) = e^(-(m/n)·(ln 2)²)
```

Take natural logs and solve for `m`:

```
ln(p) = -(m/n) · (ln 2)²   →   m = -n · ln(p) / (ln 2)²
```

Then `k = round((m/n) · ln 2)` (minimum 1). Integer truncation of `m` and rounding of `k` can push the achieved rate slightly above the target — bloomlab checks this with `TheoryFalsePositiveRate` after sizing.

`PlanSizing(n, p)` resolves `(m, k)` and reports the achieved theoretical FPR and fill fraction at capacity in one call. When you already have a `FilterConfig` (for example from CLI flags), use `PlanSizingFromFilter(fc)` so `-min-bits` and `-max-k` bounds are honored. For a numbered walkthrough with your inputs, use `FormatSizingDerivation` or:

```bash
go run ./cmd/fprcalc -n 10000 -p 0.01 -derive
```

### Worked example (`n = 10_000`, `p = 0.01`)

| Quantity | Value |
|----------|-------|
| Target capacity `n` | 10,000 |
| Target FPR `p` | 0.01 (1%) |
| Bits `m` | 95,850 (~9.6 bits/item) |
| Hash functions `k` | 6 |
| Fill fraction `f` at capacity | ≈ 0.465 (46.5%) |
| Achieved theory FPR | ≈ 0.0101 (1.01%) |

#### Substituting the numbers

With `m = 95,850` and `k = 6` from the table above:

1. **Exponent `kn/m`:** `6 × 10,000 / 95,850 ≈ 0.6260`
2. **Clear probability:** `P(clear) ≈ e^(-0.6260) ≈ 0.5347`
3. **Fill fraction:** `f ≈ 1 - 0.5347 ≈ 0.4653` (46.5%)
4. **False positive rate:** `p ≈ f^k ≈ 0.4653^6 ≈ 0.01014` (1.014%)

The achieved rate is slightly above the 1% target because `k` is rounded down from the continuous optimum `k* ≈ 6.644`. `FormatSizingDerivation` and `go run ./cmd/fprcalc -n 10000 -p 0.01 -derive` print the same steps programmatically.

Check with the library or CLI:

```bash
go run ./cmd/fprcalc -n 10000 -p 0.01
```

```go
plan, _ := bloom.PlanSizingFromFilter(bloom.TargetFilter(10_000, 0.01))
fmt.Println(plan)
// target n=10000 p=0.01 -> m=95850 (9.59 bits/item) k=6
// at capacity: fill≈0.465 (46.5%), theory FPR≈0.01014 (1.014%)
```

Probe FPR after a different insert count without rebuilding sizing:

```bash
go run ./cmd/fprcalc -n 10000 -p 0.01 -at 15000
```

Sizing fixes `m` and `k` from the target capacity; inserting more distinct keys than planned raises fill and FPR. With the same `m = 95,850` and `k = 6` but `n = 15,000`:

1. **Exponent `kn/m`:** `6 × 15,000 / 95,850 ≈ 0.9390`
2. **Fill fraction:** `f ≈ 1 - e^(-0.9390) ≈ 0.609` (60.9%)
3. **False positive rate:** `p ≈ 0.609^6 ≈ 0.0510` (5.10%)

The filter is sized for 10k keys at 1% FPR; at 15k inserts the theoretical rate is roughly five times higher. `Filter.TheoryFPR()` tracks this as keys are added — use `-at` or `TheoryFPRAt(n)` to preview overload before construction.

### Theory vs. practice

| Factor | Effect on FPR |
|--------|----------------|
| Duplicate inserts | Raise fill ratio without new membership info → higher FPR |
| `MinBits` floor | Forces larger `m` than the formula → lower FPR than target |
| `MaxHashCount` cap | Limits `k` → can exceed target FPR |
| Hash quality | Poor mixing can deviate from the independence model |
| Integer rounding | Truncating `m` and `k` can push theory slightly above `p` (≤ ~20% in tests) |

Use `TheoryFalsePositiveRate(n, m, k)` to evaluate a sizing plan, `TheoryFillFraction(n, m, k)` for expected bit density, `FilterConfig.TheoryFPRAt(n)` before construction, or `Filter.TheoryFPR()` at runtime:

```go
fc := bloom.TargetFilter(10_000, 0.01)
m, k, _ := fc.Size()
fmt.Println(bloom.TheoryFillFraction(10_000, m, k))       // ~0.47
fmt.Println(bloom.TheoryFalsePositiveRate(10_000, m, k)) // ~0.01

f, _ := bloom.NewFilterFrom(fc)
for i := 0; i < 10_000; i++ {
	f.Add([]byte(fmt.Sprintf("key:%d", i)))
}
fmt.Println(f.TheoryFPR())  // theoretical FPR at current insert count
fmt.Println(f.FillRatio())  // observed fill (may differ slightly from theory)
```

**Empirical FPR** is measured by inserting `n` distinct keys, then probing many absent keys and counting false positives: `rate = falsePositives / trials`. Tests in `bloom/fpr_empirical_test.go` assert empirical rates stay within generous bounds of theory; see also `TestFalsePositiveRate` in `bloom/bloom_test.go`.

### API reference

| Function | Returns |
|----------|---------|
| `TheoryFillFraction(n, m, k)` | Expected fill fraction `f ≈ 1 - e^(-kn/m)` |
| `TheoryFalsePositiveRate(n, m, k)` | Theory FPR `p ≈ f^k`; 0 when `n`, `m`, or `k` is zero |
| `ContinuousOptimalM(n, p)` | Real-valued `m` before truncation and bounds |
| `ContinuousOptimalK(m, n)` | Real-valued `k` before rounding and caps |
| `PlanSizing(n, p)` / `PlanSizingFromFilter(fc)` | Resolved `m`, `k`, achieved FPR, and fill at capacity |
| `FormatSizingDerivation(plan)` | Numbered derivation text (same as `fprcalc -derive`) |
| `FilterConfig.TheoryFPRAt(n)` | Theory FPR for a config at insert count `n` |
| `Filter.TheoryFPR()` | Theory FPR using the filter's current insert count |

### Invariants

Under the independence model, theory FPR satisfies:

- **Fill identity:** `TheoryFalsePositiveRate(n, m, k) = TheoryFillFraction(n, m, k)^k`
- **Monotonic in inserts:** more distinct keys → higher FPR (for fixed `m`, `k`)
- **Monotonic in bits:** larger `m` → lower FPR (for fixed `n`, `k`)
- **At continuous optimum:** `k* = (m/n)·ln 2` gives `f ≈ 1/2`, hence `p ≈ (1/2)^k*`

Property tests in `bloom/property_suite_test.go` validate these relationships with quick-check.

## Demo apps

Build and run from the repo root:

```bash
# Standard Bloom filter — add/check words
go run ./cmd/bloomdemo -n 5000 -p 0.01 hello world hello

# Counting Bloom filter — add or remove (optional murmur3 hashing)
go run ./cmd/countingdemo alpha beta
go run ./cmd/countingdemo -hash murmur3 -seed 42 alpha beta
go run ./cmd/countingdemo -remove alpha

# Generic stream deduper — classify any stdin lines as new or duplicate
printf '%s\n' 'alpha' 'beta' 'alpha' | go run ./cmd/streamdedup
printf '%s\n' 'Alpha' 'alpha' | go run ./cmd/streamdedup -ignore-case
printf '%s\n' 'log-a' 'log-b' 'log-a' | go run ./cmd/streamdedup -json
printf '%s\n' 'a' 'b' 'a' 'c' | go run ./cmd/streamdedup -novel-only
printf '%s\n' 'a' 'b' 'a' 'c' 'a' | go run ./cmd/streamdedup -dup-only
go run ./cmd/streamdedup urls.txt   # optional single file argument

# Interactive dedup demo — built-in samples or pass lines as arguments
go run ./cmd/dedupdemo
go run ./cmd/dedupdemo -sample tracking -url -normalize -strip-tracking
go run ./cmd/dedupdemo alpha beta alpha
go run ./cmd/dedupdemo -url -normalize 'https://Example.com/' 'http://example.com:80'

# Counting stream deduper — same flow with removable keys (prefix lines with -)
printf '%s\n' 'alpha' 'beta' 'alpha' '-alpha' 'alpha' | go run ./cmd/countingdedup
printf '%s\n' 'Alpha' 'alpha' | go run ./cmd/countingdedup -ignore-case
printf '%s\n' 'x' 'x' '-x' 'x' | go run ./cmd/countingdedup -json
printf '%s\n' 'a' 'b' 'a' 'c' 'a' | go run ./cmd/countingdedup -dup-only

# URL stream deduper — same check-then-insert flow with optional URL canonicalization
printf '%s\n' 'https://a.test' 'https://b.test' 'https://a.test' | go run ./cmd/urldedup
printf '%s\n' 'https://a.test' 'https://b.test' 'https://a.test' | go run ./cmd/urldedup -quiet
printf '%s\n' 'https://a.test' 'https://b.test' 'https://a.test' | go run ./cmd/urldedup -novel-only
printf '%s\n' 'https://a.test' 'https://b.test' 'https://a.test' | go run ./cmd/urldedup -dup-only

# URL dedup with canonicalization (case, ports, trailing slashes, fragments)
printf '%s\n' 'https://Example.com/' 'http://example.com:80' | go run ./cmd/urldedup -normalize
printf '%s\n' 'https://a.test' 'https://a.test' | go run ./cmd/urldedup -json

# Strip query strings, tracking params, fragments, or dedupe by host only
printf '%s\n' 'https://a.test/x?a=1' 'https://a.test/x?b=2' | go run ./cmd/urldedup -normalize -strip-query
printf '%s\n' 'https://a.test/p?utm_source=x&id=1' 'https://a.test/p?fbclid=y&id=1' | go run ./cmd/urldedup -normalize -strip-tracking
printf '%s\n' 'https://a.test/page#one' 'https://a.test/page#two' | go run ./cmd/urldedup -strip-fragment
printf '%s\n' 'https://a.test/one' 'https://a.test/two' | go run ./cmd/urldedup -normalize -domain-only

# Counting URL stream deduper — canonicalize URLs and allow removable keys (prefix lines with -)
printf '%s\n' \
  'https://a.test/page?utm_source=x' \
  'https://a.test/page?fbclid=y' \
  '-https://a.test/page' \
  'https://a.test/page' \
  | go run ./cmd/countingurldedup -normalize -strip-tracking
printf '%s\n' 'https://a.test/path/' 'https://a.test/path' '-https://a.test/path' 'https://a.test/path' | go run ./cmd/countingurldedup -normalize
printf '%s\n' 'https://a.test' 'https://b.test' 'https://a.test' | go run ./cmd/countingurldedup -dup-only

# Sizing calculator — show m, k, fill fraction, and theory FPR for a target
go run ./cmd/fprcalc -n 10000 -p 0.01
go run ./cmd/fprcalc -n 10000 -p 0.01 -derive   # step-by-step FPR math
go run ./cmd/fprcalc -n 5000 -p 0.001 -at 7500   # FPR when insert count exceeds capacity
```

## Tests & benchmarks

```bash
go test ./...
go test -bench=. -benchmem ./bloom/
```

Compare Bloom filter throughput and memory against a `map[string]struct{}` hash set:

```bash
# Throughput: Add, Contains (hit/miss)
go test -bench='Filter|MapSet' -benchmem ./bloom/

# Space: storage-bytes/item at 100k inserts, 1% target FPR
go test -bench=Footprint -benchmem ./bloom/
```

Bloom filters trade exact membership and per-insert heap allocations for a fixed bit slice; hash sets allocate per key but offer O(1) exact lookups with lower constant factors.

### dedup subsystem

The `dedup` package implements the check-then-insert pattern shared by the stream dedup CLIs: classify each line as novel or duplicate, optionally emit results, and print filter stats on stderr. `Classifier` wraps a standard Bloom filter; `CountingClassifier` adds `Remove` for sliding-window workloads where keys leave the set.

```go
f, _ := bloom.NewFilter(bloom.TargetConfig(10_000, 0.01))
c := dedup.NewClassifier(f, dedup.TrimKey)
dup, ok := c.Classify("user:42") // first sight → dup=false, ok=true

cf, _ := bloom.NewCountingFilter(bloom.TargetConfig(10_000, 0.01))
cc := dedup.NewCountingClassifier(cf, nil)
_, _, _ = cc.Classify("session-a")
cc.Remove("session-a") // key can be seen again as novel
```

The `countingdedup` CLI reads stdin and treats lines prefixed with `-` (override with `-remove-prefix`) as removals instead of classifications.

`dedupdemo` is the quickest way to try stream or URL dedup: run with no input to execute a built-in sample (`-sample stream|url|tracking`), pass lines as arguments, or pipe stdin. The stream dedup CLIs (`streamdedup`, `urldedup`, `countingdedup`, `countingurldedup`) accept an optional single file path argument in addition to stdin.

### benchcompare subsystem

The `benchcompare` package runs paired workloads (add, contains hit/miss/mixed, mixed stream dedup, and counting-filter remove) against a Bloom filter and a hash set, then reports throughput, bytes-per-item, and heap allocations side by side:

```bash
# Full comparison table at default sizing (100k items, 1% FPR)
go run ./cmd/benchcompare

# Smaller run for quick local checks
go run ./cmd/benchcompare -n 10000 -repeats 2

# Sweep false positive targets to see Bloom sizing trade-offs (hash set unchanged)
go run ./cmd/benchcompare -sweep-fpr -n 50000
go run ./cmd/benchcompare -sweep-fpr -p-values 0.001,0.01,0.05,0.1 -markdown

# Compare hash strategies (Bloom only; hash set ignores -hash/-seed)
go run ./cmd/benchcompare -hash murmur3 -seed 42 -n 10000

# Sweep hash families to compare Bloom add throughput at fixed sizing
go run ./cmd/benchcompare -sweep-hash -n 50000
go run ./cmd/benchcompare -sweep-hash -hash-values fnv,murmur3,xxhash,wyhash -markdown

# Sweep item counts to see how hash set footprint scales vs fixed Bloom sizing
go run ./cmd/benchcompare -sweep-size
go run ./cmd/benchcompare -sweep-size -size-values 10000,100000,1000000 -p 0.01 -markdown

# Sweep lookup hit ratios to compare bloom vs hash set on mixed contains workloads
go run ./cmd/benchcompare -sweep-mix -n 50000 -repeats 2
go run ./cmd/benchcompare -sweep-mix -mix-values 0,0.25,0.5,0.75,1 -markdown

# Sweep key byte lengths to see hash set footprint grow while Bloom stays fixed
go run ./cmd/benchcompare -sweep-keylen -n 10000
go run ./cmd/benchcompare -sweep-keylen -keylen-values 16,64,256,1024 -markdown

# Markdown table for docs or CI artifacts
go run ./cmd/benchcompare -markdown > docs/benchcompare.md
```

Programmatic use:

```go
results, _ := benchcompare.Compare(benchcompare.DefaultConfig())
fmt.Print(benchcompare.FormatReport(benchcompare.DefaultConfig(), results))
```

Go benchmarks in `benchcompare/` reuse the same measurement helpers:

```bash
go test -bench=. -benchmem ./benchcompare/
go test -bench=ReportMetrics ./benchcompare/
```

## API overview

| Type | Add | Contains | Remove | Notes |
|------|-----|----------|--------|-------|
| `Filter` | ✓ | ✓ | — | Classic bit-slice Bloom filter |
| `CountingFilter` | ✓ | ✓ | ✓ | 8-bit counters by default; `WithCounterWidth(2\|4\|16\|32\|64)` for packed or wider variants; overflow at counter max; `Clear`, `CounterBytes`, `CounterWidth`, `ApproximateCount`, `TheoryFPR`, `FillRatio` |

### Configuration

Configuration separates **sizing** (how many bits and hash functions) from **hashing** (which hash family and seed). The structured `FilterConfig` type makes that split explicit:

```go
fc, err := bloom.BuildFilterConfig(bloom.SizingTarget, bloom.TargetSpec{
	Capacity: 10_000,
	FPR:      0.01,
	Bounds:   bloom.SizingBounds{MinBits: 256},
}, bloom.ExplicitSpec{},
)
fc = fc.Apply(bloom.WithFilterHash(bloom.HashMurmur3))
f, err := bloom.NewFilterFrom(fc)
```

`SizingConfig` uses a discriminated `Mode` field (`SizingTarget` or `SizingExplicit`) so target and explicit inputs never share the same flat struct. Counting filters add counter width via `CountingConfig`:

```go
cc, err := bloom.BuildCountingConfig(bloom.SizingExplicit, bloom.TargetSpec{},
	bloom.ExplicitSpec{Bits: 1024, HashCount: 4},
	bloom.WithCountingCounterWidth(16),
)
cf, err := bloom.NewCountingFilterFrom(cc)
```

The legacy `bloom.Config` type remains available and converts cleanly: `cfg.FilterConfig()` and `fc.Config()` round-trip sizing, hash, and counter settings. Existing helpers like `TargetConfig` and `NewFilter(cfg)` continue to work.

Sizing mode on either type: call `Mode()` for `SizingTarget` or `SizingExplicit`, or use `Target()` / `Explicit()` on `Config` to read typed inputs. `Bounds()` returns target-mode limits (`SizingBounds` with package defaults via `Resolved()`).

| Helper | Use when |
|--------|----------|
| `TargetFilter(n, p, opts...)` / `ExplicitFilter(m, k, opts...)` | Structured config without validation |
| `BuildFilterConfig(mode, target, explicit, opts...)` | Validate typed specs and return `FilterConfig` |
| `TargetCounting` / `ExplicitCounting` / `BuildCountingConfig` | Counting filter configuration |
| `NewFilterFrom(fc)` / `NewCountingFilterFrom(cc)` | Construct filters from structured config |
| `FilterConfig.Config()` / `Config.FilterConfig()` | Convert between structured and legacy forms |
| `TargetConfig(n, p, opts...)` | Legacy: derive `m` and `k` from capacity and FPR |
| `ExplicitConfig(m, k, opts...)` | Legacy: fix bit count and hash functions directly |
| `BuildConfig(mode, target, explicit, opts...)` | Legacy validated builder returning `Config` |
| `WithFilterHash` / `WithFilterSeed` / `WithFilterSizingBounds` | Options on `FilterConfig` |
| `WithCountingCounterWidth` / `WithCountingHash` | Options on `CountingConfig` |
| `Config.Apply(opts...)` | Apply legacy construction options to an existing config copy |
| `WithHash(strategy)` / `WithSeed(seed)` / `WithHashConfig(h)` | Legacy hash options at construction |
| `Config.WithSeed` / `WithHash` / `WithSizingBounds` / `WithCounterWidth` | Immutable copy updates on `Config` |
| `NewFilter(cfg)` / `NewCountingFilter(cfg)` | Legacy constructors (delegate to structured config internally) |
| `New(n, p)` / `NewCountingFromTarget(n, p)` | Shorthand for target sizing |

CLIs share filter flags via `cmd/internal/filterflags`: target sizing uses `-n`/`-p`; explicit sizing uses `-m`/`-k`. Demos and tools resolve flags through `FilterConfig()` / `CountingConfig()` and construct filters with `NewFilterFrom` / `NewCountingFilterFrom`. The legacy `Config()` helper remains for round-trip compatibility. Stream dedup tools share output flags via `cmd/internal/streamflags` (`-quiet`, `-json`, `-ignore-case`, `-novel-only`, `-dup-only`).

Target sizing uses:

- `m = -n · ln(p) / (ln 2)²`
- `k = (m/n) · ln 2`

See [False positive rate](#false-positive-rate) for the full `p ≈ (1 - e^(-kn/m))^k` derivation and practical caveats.

### Hashing

Bit positions use **double hashing**: `h(i) = (h1 + i·h2) mod m`. Hash settings live in `Config.Hash` and can be supplied via `WithHash` and `WithSeed`:

| Strategy | Name | Notes |
|----------|------|-------|
| `HashFNV` | `fnv` | Default; FNV-1a 64-bit (backward compatible; seed ignored) |
| `HashMurmur3` | `murmur3` | MurmurHash3 x64_128 single-pass derivation |
| `HashXXHash` | `xxhash` | Single-pass xxHash-128-style derivation |
| `HashWyhash` | `wyhash` | wyhash single-pass paired-state derivation |
| `HashHighway` | `highway` | HighwayHash-128 single-pass derivation (seed-sensitive; keyed PRF) |

```go
f, _ := bloom.NewFilter(bloom.TargetConfig(10_000, 0.01,
	bloom.WithHash(bloom.HashMurmur3),
	bloom.WithSeed(42),
))
```

Compare hash uniformity before picking a strategy:

```go
spread := bloom.MeasureBucketSpread(cfg.Hasher(), m, k, 10_000, func(i int) []byte {
	return []byte(fmt.Sprintf("key-%d", i))
})
fmt.Println(spread.ChiSquared, spread.WithinSpreadTolerance(4))
best := bloom.BestUniformStrategy(m, k, 10_000, keyFor, bloom.AllStrategies())
```

Tune seeds and pick a strategy for a planned layout:

```go
opts := bloom.TuneOptions{
	M: m, K: k, Samples: 10_000,
	KeyFor: bloom.KeyForDistribution(bloom.KeyURL, "probe"),
}
report := bloom.RecommendHasher(opts, bloom.AllStrategies(), bloom.DefaultTuneSeeds())
fmt.Println(report.Best.Strategy, report.Best.Seed, report.Best.Spread.ChiSquared)
```

Tuning reports also include **double-hash stride** (`gcd(h2, m)` defects) and **h1/h2 correlation** (`|r|` near zero is better). Apply the recommendation at construction time:

```go
cfg, err := bloom.TargetConfig(10_000, 0.01).WithRecommendedHash(bloom.RecommendedHashOptions{
	Samples:      5000,
	Distribution: bloom.KeyURL,
	KeyPrefix:    "probe",
})
f, _ := bloom.NewFilter(cfg)
```

Or as a config option (panics if tuning inputs are invalid):

```go
cfg := bloom.TargetConfig(10_000, 0.01, bloom.WithRecommendedHash(bloom.RecommendedHashOptions{
	Samples: 5000,
}))
```

The `hashtune` CLI compares strategies and seeds for a target layout. Use `-key-dist` to probe keys that resemble your workload (URLs, UUIDs, fixed-width binary IDs), or `-key-file` to read sample keys from a text file:

```bash
go run ./cmd/hashtune -n 50000 -p 0.01 -key-dist url
go run ./cmd/hashtune -n 10000 -key-file sample-urls.txt -samples 5000
go run ./cmd/hashtune -strategy murmur3 -key-dist uuid -seeds 0,42,0xdeadbeef
```

When `m` is a power of two, indexing uses a bitmask fast path instead of modulo. Changing strategy or seed changes bit positions — filters are not interoperable across hash settings.

Demo CLIs accept `-hash fnv|murmur3|xxhash|wyhash|highway` and `-seed <uint64>`.

## License

MIT
