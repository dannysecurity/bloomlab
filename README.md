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

```go
package main

import (
	"fmt"
	"github.com/dannysecurity/bloomlab/bloom"
)

func main() {
	// Expect ~10k items, ~1% false positive rate
	f, _ := bloom.New(10_000, 0.01)
	f.Add([]byte("user:42"))
	fmt.Println(f.Contains([]byte("user:42"))) // true
}
```

### Counting variant (with deletion)

```go
cf, _ := bloom.NewCountingFromTarget(10_000, 0.01)
_ = cf.Add([]byte("session:abc"))
cf.Remove([]byte("session:abc"))
fmt.Println(cf.Contains([]byte("session:abc"))) // false
```

## Demo apps

Build and run from the repo root:

```bash
# Standard Bloom filter — add/check words
go run ./cmd/bloomdemo -n 5000 -p 0.01 hello world hello

# Counting Bloom filter — add or remove
go run ./cmd/countingdemo alpha beta
go run ./cmd/countingdemo -remove alpha
```

## Tests & benchmarks

```bash
go test ./...
go test -bench=. -benchmem ./bloom/
```

## API overview

| Type | Add | Contains | Remove | Notes |
|------|-----|----------|--------|-------|
| `Filter` | ✓ | ✓ | — | Classic bit-slice Bloom filter |
| `CountingFilter` | ✓ | ✓ | ✓ | 8-bit counters; overflow at 255; `FillRatio` for occupancy |

Sizing uses the standard formulas:

- `m = -n · ln(p) / (ln 2)²`
- `k = (m/n) · ln 2`

Hashing uses FNV-1a double hashing to derive `k` bit positions.

## License

MIT
