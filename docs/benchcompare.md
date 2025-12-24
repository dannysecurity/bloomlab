# Bloom filter vs hash set benchmarks

Sample output from `benchcompare` at `n=10000`, `p=0.01`, lookup repeats `2`. Regenerate with:

```bash
go run ./cmd/benchcompare -n 10000 -repeats 2 -markdown > docs/benchcompare.md
go run ./cmd/benchcompare -sweep-mix -n 10000 -repeats 2 -markdown >> docs/benchcompare.md
```

## Full scenario comparison

Configuration: `n=10000`, `p=0.0100`, lookup repeats `2`.

| Scenario | Bloom ns/op | Hash set ns/op | Speedup | Bloom B/item | Hash set B/item | Space ratio | Bloom allocs/op | Hash set allocs/op | Alloc ratio | Notes |
|----------|-------------|----------------|---------|--------------|-----------------|-------------|-----------------|--------------------|-------------|-------|
| add | 24 | 118 | 4.96x | 1.2 | 39.9 | 33.3x | 0.00 | 1.00 | 3334.3x | theory FPR 1.014% |
| contains_hit | 21 | 25 | 1.19x | 1.2 | 32.0 | 26.7x | 0.00 | 0.00 | 0.0x | theory FPR 1.014% |
| contains_miss | 31 | 9 | 0.30x | 1.2 | 32.0 | 26.7x | 0.00 | 0.00 | 0.0x | theory FPR 1.014% |
| contains_mixed | 26 | 16 | 0.60x | 1.2 | 32.0 | 26.7x | 0.00 | 0.00 | 0.0x | hit ratio 50%; theory FPR 1.014% |
| mixed_stream | 35 | 50 | 1.41x | 2.4 | 32.8 | 13.7x | 0.00 | 1.00 | 3334.0x | dup calls bloom=5002 hashset=5000 |
| remove | 48 | 37 | 0.77x | 9.6 | 32.0 | 3.3x | 0.00 | 1.00 | 3334.0x | counting bloom filter |

## Lookup mix sweep

Contains-mixed workload at `n=10000`, `p=0.0100`, lookup repeats `2` across hit ratios.

| Hit ratio | Bloom ns/op | Hash set ns/op | Speedup | Bloom B/item | Hash set B/item | Space ratio | Bloom allocs/op | Hash set allocs/op | Alloc ratio |
|-----------|-------------|----------------|---------|--------------|-----------------|-------------|-----------------|--------------------|-------------|
| 0% | 31 | 9 | 0.30x | 1.2 | 32.0 | 26.7x | 0.00 | 0.00 | 0.0x |
| 25% | 30 | 12 | 0.40x | 1.2 | 32.0 | 26.7x | 0.00 | 0.00 | 0.0x |
| 50% | 28 | 16 | 0.56x | 1.2 | 32.0 | 26.7x | 0.00 | 0.00 | 0.0x |
| 75% | 24 | 21 | 0.89x | 1.2 | 32.0 | 26.7x | 0.00 | 0.00 | 0.0x |
| 100% | 21 | 25 | 1.14x | 1.2 | 32.0 | 26.7x | 0.00 | 0.00 | 0.0x |

At low hit ratios, hash set lookups dominate because absent keys exit after the first probe; Bloom filters still scan all `k` bit positions on every lookup.
