# bloomlab

A Go toolkit for [Bloom filters](https://en.wikipedia.org/wiki/Bloom_filter): space-efficient probabilistic membership sets with configurable false-positive rates.

## Features

- **Standard Bloom filter** (`bloom.Filter`) — fixed-size bit array, optimal `m` and `k` from target capacity and FPR
- **Counting Bloom filter** (`bloom.CountingFilter`) — supports deletion via per-bit counters
- **Benchmarks** — add, lookup, and remove throughput
- **Demo CLIs** — `bloomdemo` and `countingdemo` for interactive exploration

## Install

```bash
go get github.com/dannysecurity/bloomlab
```

## Quick start

Configuration is expressed through `bloom.Config`, which both filter types share:

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

There are no false negatives: if a key was inserted, `Contains` always returns true.

### Sizing formulas

Given target capacity `n` and desired FPR upper bound `p`, bloomlab picks `m` (bits) and `k` (hash functions) to minimize space while meeting the budget:

- `m = -n · ln(p) / (ln 2)²`
- `k = (m/n) · ln 2` (rounded, with a minimum of 1)

This `(m, k)` pair is optimal for fixed `n` and `m`: the achieved rate is close to `(1/2)^k`. The formulas appear in `optimalM` / `optimalK` inside `bloom/config.go`.

### Theory vs. practice

| Factor | Effect on FPR |
|--------|----------------|
| Duplicate inserts | Raise fill ratio without new membership info → higher FPR |
| `MinBits` floor | Forces larger `m` than the formula → lower FPR than target |
| `MaxHashCount` cap | Limits `k` → can exceed target FPR |
| Hash quality | Poor mixing can deviate from the independence model |

Use `TheoryFalsePositiveRate(n, m, k)` to evaluate a sizing plan, `Config.TheoryFPRAt(n)` before construction, or `Filter.TheoryFPR()` at runtime:

```go
cfg := bloom.TargetConfig(10_000, 0.01)
m, k, _ := cfg.Size()
fmt.Println(bloom.TheoryFalsePositiveRate(10_000, m, k)) // ~0.01

f, _ := bloom.NewFilter(cfg)
for i := 0; i < 10_000; i++ {
	f.Add([]byte(fmt.Sprintf("key:%d", i)))
}
fmt.Println(f.TheoryFPR()) // theoretical FPR at current insert count
```

Empirical validation lives in `TestFalsePositiveRate` (`bloom/bloom_test.go`).

## Demo apps

Build and run from the repo root:

```bash
# Standard Bloom filter — add/check words
go run ./cmd/bloomdemo -n 5000 -p 0.01 hello world hello

# Counting Bloom filter — add or remove (optional murmur3 hashing)
go run ./cmd/countingdemo alpha beta
go run ./cmd/countingdemo -hash murmur3 -seed 42 alpha beta
go run ./cmd/countingdemo -remove alpha
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

## API overview

| Type | Add | Contains | Remove | Notes |
|------|-----|----------|--------|-------|
| `Filter` | ✓ | ✓ | — | Classic bit-slice Bloom filter |
| `CountingFilter` | ✓ | ✓ | ✓ | 8-bit counters; overflow at 255; `ApproximateCount`, `TheoryFPR`, `FillRatio` |

### Configuration

| Helper | Use when |
|--------|----------|
| `TargetConfig(n, p)` | Derive `m` and `k` from expected capacity and FPR |
| `ExplicitConfig(m, k)` | Fix bit count and hash functions directly |
| `TheoryFalsePositiveRate(n, m, k)` | Theoretical FPR after `n` inserts |
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

Bit positions use **double hashing**: `h(i) = (h1 + i·h2) mod m`. The package provides pluggable hash strategies via `Config.HashStrategy` and `Config.HashSeed`:

| Strategy | Name | Notes |
|----------|------|-------|
| `HashFNV` | `fnv` | Default; FNV-1a 64-bit (backward compatible) |
| `HashMurmur3` | `murmur3` | MurmurHash3 64-bit with independent seeds for `h1`/`h2` |

```go
cfg := bloom.TargetConfig(10_000, 0.01)
cfg.HashStrategy = bloom.HashMurmur3
cfg.HashSeed = 42
f, _ := bloom.NewFilter(cfg)
```

When `m` is a power of two, indexing uses a bitmask fast path instead of modulo. Changing strategy or seed changes bit positions — filters are not interoperable across hash settings.

Demo CLIs accept `-hash fnv|murmur3` and `-seed <uint64>`.

## License

MIT
