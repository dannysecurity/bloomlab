# bloomlab

A Go toolkit for [Bloom filters](https://en.wikipedia.org/wiki/Bloom_filter): space-efficient probabilistic membership sets with configurable false-positive rates.

## Features

- **Standard Bloom filter** (`bloom.Filter`) — fixed-size bit array, optimal `m` and `k` from target capacity and FPR
- **Counting Bloom filter** (`bloom.CountingFilter`) — supports deletion via per-bit counters
- **Benchmarks** — add, lookup, and remove throughput
- **benchcompare** — programmatic Bloom filter vs `map[string]struct{}` comparison with CLI and Go benchmarks
- **Demo CLIs** — `bloomdemo`, `countingdemo`, `urldedup`, `fprcalc`, and `benchcompare` for interactive exploration

## Install

```bash
go get github.com/dannysecurity/bloomlab
```

## Quick start

Configuration is expressed through `bloom.Config`, which both filter types share. Sizing and hashing are configured separately — hash settings do not affect `m` or `k`:

```go
package main

import (
	"fmt"
	"github.com/dannysecurity/bloomlab/bloom"
)

func main() {
	cfg := bloom.TargetConfig(10_000, 0.01) // ~10k items, ~1% FPR
	f, _ := bloom.NewFilter(cfg)
	f.Add([]byte("user:42"))
	fmt.Println(f.Contains([]byte("user:42"))) // true
	fmt.Println(cfg.String())                  // target n=10000 p=0.01 -> m=... k=...
}
```

Pass hash settings as options instead of mutating the config after construction:

```go
cfg := bloom.TargetConfig(10_000, 0.01,
	bloom.WithHash(bloom.HashMurmur3),
	bloom.WithSeed(42),
)
f, _ := bloom.NewFilter(cfg)
```

Shorthand constructors remain available:

```go
f, _ := bloom.New(10_000, 0.01)
```

### Counting variant (with deletion)

```go
cfg := bloom.TargetConfig(10_000, 0.01)
cf, _ := bloom.NewCountingFilter(cfg)
_ = cf.Add([]byte("session:abc"))
cf.Remove([]byte("session:abc"))
fmt.Println(cf.Contains([]byte("session:abc"))) // false
fmt.Println(cf.TheoryFPR())                       // theoretical FPR at current insert count
```

For fixed sizing, use explicit configuration:

```go
cf, _ := bloom.NewCountingFilter(bloom.ExplicitConfig(1024, 4))
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

This `(m, k)` pair is optimal for fixed `n` and `m`: the achieved rate is close to `(1/2)^k`. The formulas appear in `optimalM` / `optimalK` inside `bloom/config.go`.

To derive `m`: substitute the continuous optimum `k* = (m/n) ln 2` into `p ≈ (1/2)^(m/n ln 2)` and solve for `m`.

`PlanSizing(n, p)` resolves `(m, k)` and reports the achieved theoretical FPR and fill fraction at capacity in one call.

### Worked example (`n = 10_000`, `p = 0.01`)

| Quantity | Value |
|----------|-------|
| Target capacity `n` | 10,000 |
| Target FPR `p` | 0.01 (1%) |
| Bits `m` | 95,850 (~9.6 bits/item) |
| Hash functions `k` | 6 |
| Fill fraction `f` at capacity | ≈ 0.465 (46.5%) |
| Achieved theory FPR | ≈ 0.0101 (1.01%) |

Check with the library or CLI:

```bash
go run ./cmd/fprcalc -n 10000 -p 0.01
```

```go
plan, _ := bloom.PlanSizing(10_000, 0.01)
fmt.Println(plan)
// target n=10000 p=0.01 -> m=95850 (9.59 bits/item) k=6
// at capacity: fill≈0.465 (46.5%), theory FPR≈0.01014 (1.014%)
```

Probe FPR after a different insert count without rebuilding sizing:

```bash
go run ./cmd/fprcalc -n 10000 -p 0.01 -at 15000
```

### Theory vs. practice

| Factor | Effect on FPR |
|--------|----------------|
| Duplicate inserts | Raise fill ratio without new membership info → higher FPR |
| `MinBits` floor | Forces larger `m` than the formula → lower FPR than target |
| `MaxHashCount` cap | Limits `k` → can exceed target FPR |
| Hash quality | Poor mixing can deviate from the independence model |
| Integer rounding | Truncating `m` and `k` can push theory slightly above `p` (≤ ~20% in tests) |

Use `TheoryFalsePositiveRate(n, m, k)` to evaluate a sizing plan, `TheoryFillFraction(n, m, k)` for expected bit density, `Config.TheoryFPRAt(n)` before construction, or `Filter.TheoryFPR()` at runtime:

```go
cfg := bloom.TargetConfig(10_000, 0.01)
m, k, _ := cfg.Size()
fmt.Println(bloom.TheoryFillFraction(10_000, m, k))       // ~0.47
fmt.Println(bloom.TheoryFalsePositiveRate(10_000, m, k)) // ~0.01

f, _ := bloom.NewFilter(cfg)
for i := 0; i < 10_000; i++ {
	f.Add([]byte(fmt.Sprintf("key:%d", i)))
}
fmt.Println(f.TheoryFPR())  // theoretical FPR at current insert count
fmt.Println(f.FillRatio())  // observed fill (may differ slightly from theory)
```

**Empirical FPR** is measured by inserting `n` distinct keys, then probing many absent keys and counting false positives: `rate = falsePositives / trials`. Tests in `bloom/fpr_empirical_test.go` assert empirical rates stay within generous bounds of theory; see also `TestFalsePositiveRate` in `bloom/bloom_test.go`.

## Demo apps

Build and run from the repo root:

```bash
# Standard Bloom filter — add/check words
go run ./cmd/bloomdemo -n 5000 -p 0.01 hello world hello

# Counting Bloom filter — add or remove (optional murmur3 hashing)
go run ./cmd/countingdemo alpha beta
go run ./cmd/countingdemo -hash murmur3 -seed 42 alpha beta
go run ./cmd/countingdemo -remove alpha

# Stream deduper — classify stdin lines as new or duplicate (URLs, logs, etc.)
printf '%s\n' 'https://a.test' 'https://b.test' 'https://a.test' | go run ./cmd/urldedup
printf '%s\n' 'https://a.test' 'https://b.test' 'https://a.test' | go run ./cmd/urldedup -quiet

# URL dedup with canonicalization (case, ports, trailing slashes, fragments)
printf '%s\n' 'https://Example.com/' 'http://example.com:80' | go run ./cmd/urldedup -normalize

# Sizing calculator — show m, k, fill fraction, and theory FPR for a target
go run ./cmd/fprcalc -n 10000 -p 0.01
go run ./cmd/fprcalc -n 5000 -p 0.001 -at 7500
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

### benchcompare subsystem

The `benchcompare` package runs paired workloads (add, contains hit/miss, mixed stream dedup, and counting-filter remove) against a Bloom filter and a hash set, then reports throughput, bytes-per-item, and heap allocations side by side:

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
go run ./cmd/benchcompare -sweep-hash -hash-values fnv,murmur3,xxhash -markdown

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
| `CountingFilter` | ✓ | ✓ | ✓ | 8-bit counters; overflow at 255; `Clear`, `CounterBytes`, `ApproximateCount`, `TheoryFPR`, `FillRatio` |

### Configuration

| Helper | Use when |
|--------|----------|
| `TargetConfig(n, p, opts...)` | Derive `m` and `k` from expected capacity and FPR |
| `ExplicitConfig(m, k, opts...)` | Fix bit count and hash functions directly |
| `WithHash(strategy)` / `WithSeed(seed)` / `WithHashConfig(h)` | Set hash family, seed, or full hash config |
| `WithMinBits(m)` / `WithMaxHashCount(k)` | Bound derived sizing on target configs |
| `HashConfig` | Hash-only settings (`Strategy`, `Seed`); embedded in `Config.Hash` |
| `TheoryFalsePositiveRate(n, m, k)` | Theoretical FPR after `n` inserts |
| `TheoryFillFraction(n, m, k)` | Expected fraction of bits set after `n` inserts |
| `PlanSizing(n, p, opts...)` | Resolve `(m, k)` and report achieved theory FPR at capacity |
| `Config.TheoryFPRAt(n)` | FPR for a config at a given insert count |
| `Filter.TheoryFPR()` / `CountingFilter.TheoryFPR()` | FPR at current insert count |
| `Config.Validate()` / `Config.Size()` | Inspect or resolve sizing before construction |
| `NewFilter(cfg)` / `NewCountingFilter(cfg)` | Primary constructors |
| `New(n, p)` / `NewCountingFromTarget(n, p)` | Legacy shorthands for target sizing |

Target sizing uses:

- `m = -n · ln(p) / (ln 2)²`
- `k = (m/n) · ln 2`

See [False positive rate](#false-positive-rate) for the full `p ≈ (1 - e^(-kn/m))^k` derivation and practical caveats.

### Hashing

Bit positions use **double hashing**: `h(i) = (h1 + i·h2) mod m`. Hash settings live in `Config.Hash` and can be supplied via `WithHash` and `WithSeed`:

| Strategy | Name | Notes |
|----------|------|-------|
| `HashFNV` | `fnv` | Default; FNV-1a 64-bit (backward compatible; seed ignored) |
| `HashMurmur3` | `murmur3` | MurmurHash3 64-bit with independent seeds for `h1`/`h2` |
| `HashXXHash` | `xxhash` | xxHash 64-bit with independent seeds for `h1`/`h2` |

```go
f, _ := bloom.NewFilter(bloom.TargetConfig(10_000, 0.01,
	bloom.WithHash(bloom.HashMurmur3),
	bloom.WithSeed(42),
))
```

When `m` is a power of two, indexing uses a bitmask fast path instead of modulo. Changing strategy or seed changes bit positions — filters are not interoperable across hash settings.

Demo CLIs accept `-hash fnv|murmur3|xxhash` and `-seed <uint64>`.

## License

MIT
