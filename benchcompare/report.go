package benchcompare

import (
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/dannysecurity/bloomlab/bloom"
)

func configHeader(cfg Config) string {
	hashNote := ""
	if cfg.Hash.Strategy != 0 || cfg.Hash.Seed != 0 {
		hashNote = fmt.Sprintf(", hash=%s", cfg.Hash.String())
	}
	return fmt.Sprintf("Bloom filter vs hash set (n=%d, p=%.4f, lookup-repeats=%d%s)",
		cfg.ItemCount, cfg.FalsePositiveRate, cfg.LookupRepeats, hashNote)
}

// FormatReport renders comparisons as a fixed-width table suitable for stdout.
func FormatReport(cfg Config, results []Comparison) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n\n", configHeader(cfg))

	tw := tabwriter.NewWriter(&b, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "SCENARIO\tBLOOM ns/op\tHASHSET ns/op\tSPEEDUP\tBLOOM B/item\tHASHSET B/item\tSPACE\tBLOOM allocs/op\tHASHSET allocs/op\tALLOCS\tNOTES")
	for _, cmp := range results {
		notes := formatNotes(cmp)
		fmt.Fprintf(tw, "%s\t%.0f\t%.0f\t%.2fx\t%.1f\t%.1f\t%.1fx\t%.2f\t%.2f\t%.1fx\t%s\n",
			cmp.Scenario,
			cmp.Bloom.NsPerOp,
			cmp.HashSet.NsPerOp,
			cmp.SpeedRatio(),
			cmp.Bloom.BytesPerItem,
			cmp.HashSet.BytesPerItem,
			cmp.SpaceRatio(),
			cmp.Bloom.AllocsPerOp,
			cmp.HashSet.AllocsPerOp,
			cmp.AllocRatio(),
			notes,
		)
	}
	_ = tw.Flush()

	b.WriteString("\nSpeedup > 1 means Bloom was faster; space > 1 means Bloom used less memory per item; allocs > 1 means Bloom allocated less per op.\n")
	return b.String()
}

// FormatMarkdown renders comparisons as a GitHub-flavored markdown table.
func FormatMarkdown(cfg Config, results []Comparison) string {
	var b strings.Builder
	fmt.Fprintf(&b, "## Bloom filter vs hash set\n\n")
	hashNote := ""
	if cfg.Hash.Strategy != 0 || cfg.Hash.Seed != 0 {
		hashNote = fmt.Sprintf(", hash `%s`", cfg.Hash.String())
	}
	fmt.Fprintf(&b, "Configuration: `n=%d`, `p=%.4f`, lookup repeats `%d`%s.\n\n",
		cfg.ItemCount, cfg.FalsePositiveRate, cfg.LookupRepeats, hashNote)
	fmt.Fprintln(&b, "| Scenario | Bloom ns/op | Hash set ns/op | Speedup | Bloom B/item | Hash set B/item | Space ratio | Bloom allocs/op | Hash set allocs/op | Alloc ratio | Notes |")
	fmt.Fprintln(&b, "|----------|-------------|----------------|---------|--------------|-----------------|-------------|-----------------|--------------------|-------------|-------|")
	for _, cmp := range results {
		notes := formatNotes(cmp)
		fmt.Fprintf(&b, "| %s | %.0f | %.0f | %.2fx | %.1f | %.1f | %.1fx | %.2f | %.2f | %.1fx | %s |\n",
			cmp.Scenario,
			cmp.Bloom.NsPerOp,
			cmp.HashSet.NsPerOp,
			cmp.SpeedRatio(),
			cmp.Bloom.BytesPerItem,
			cmp.HashSet.BytesPerItem,
			cmp.SpaceRatio(),
			cmp.Bloom.AllocsPerOp,
			cmp.HashSet.AllocsPerOp,
			cmp.AllocRatio(),
			notes,
		)
	}
	return b.String()
}

// FormatFPRSweep renders add-scenario comparisons across false positive targets.
func FormatFPRSweep(cfg Config, rates []float64, results []Comparison) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Bloom filter vs hash set — FPR sweep (n=%d, add workload)\n\n", cfg.ItemCount)

	tw := tabwriter.NewWriter(&b, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "TARGET p\tBLOOM B/item\tHASHSET B/item\tSPACE\tBLOOM ns/op\tHASHSET ns/op\tSPEEDUP\tBLOOM allocs/op\tHASHSET allocs/op\tALLOCS")
	for i, cmp := range results {
		fmt.Fprintf(tw, "%.4f\t%.1f\t%.1f\t%.1fx\t%.0f\t%.0f\t%.2fx\t%.2f\t%.2f\t%.1fx\n",
			rates[i],
			cmp.Bloom.BytesPerItem,
			cmp.HashSet.BytesPerItem,
			cmp.SpaceRatio(),
			cmp.Bloom.NsPerOp,
			cmp.HashSet.NsPerOp,
			cmp.SpeedRatio(),
			cmp.Bloom.AllocsPerOp,
			cmp.HashSet.AllocsPerOp,
			cmp.AllocRatio(),
		)
	}
	_ = tw.Flush()

	b.WriteString("\nLower p sizes the Bloom filter larger (more bits, lower FPR). Hash set footprint is unchanged.\n")
	return b.String()
}

// FormatFPRSweepMarkdown renders the FPR sweep as a markdown table.
func FormatFPRSweepMarkdown(cfg Config, rates []float64, results []Comparison) string {
	var b strings.Builder
	fmt.Fprintf(&b, "## Bloom filter vs hash set — FPR sweep\n\n")
	fmt.Fprintf(&b, "Add workload at `n=%d` across target false positive rates.\n\n", cfg.ItemCount)
	fmt.Fprintln(&b, "| Target p | Bloom B/item | Hash set B/item | Space ratio | Bloom ns/op | Hash set ns/op | Speedup | Bloom allocs/op | Hash set allocs/op | Alloc ratio |")
	fmt.Fprintln(&b, "|----------|--------------|-----------------|-------------|-------------|----------------|---------|-----------------|--------------------|-------------|")
	for i, cmp := range results {
		fmt.Fprintf(&b, "| %.4f | %.1f | %.1f | %.1fx | %.0f | %.0f | %.2fx | %.2f | %.2f | %.1fx |\n",
			rates[i],
			cmp.Bloom.BytesPerItem,
			cmp.HashSet.BytesPerItem,
			cmp.SpaceRatio(),
			cmp.Bloom.NsPerOp,
			cmp.HashSet.NsPerOp,
			cmp.SpeedRatio(),
			cmp.Bloom.AllocsPerOp,
			cmp.HashSet.AllocsPerOp,
			cmp.AllocRatio(),
		)
	}
	return b.String()
}

// FormatHashSweep renders add-scenario comparisons across hash strategies.
func FormatHashSweep(cfg Config, strategies []bloom.Strategy, results []Comparison) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Bloom filter vs hash set — hash sweep (n=%d, p=%.4f, add workload)\n\n",
		cfg.ItemCount, cfg.FalsePositiveRate)

	tw := tabwriter.NewWriter(&b, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "HASH\tBLOOM ns/op\tHASHSET ns/op\tSPEEDUP\tBLOOM B/item\tHASHSET B/item\tSPACE\tBLOOM allocs/op\tHASHSET allocs/op\tALLOCS")
	for i, cmp := range results {
		fmt.Fprintf(tw, "%s\t%.0f\t%.0f\t%.2fx\t%.1f\t%.1f\t%.1fx\t%.2f\t%.2f\t%.1fx\n",
			strategies[i].String(),
			cmp.Bloom.NsPerOp,
			cmp.HashSet.NsPerOp,
			cmp.SpeedRatio(),
			cmp.Bloom.BytesPerItem,
			cmp.HashSet.BytesPerItem,
			cmp.SpaceRatio(),
			cmp.Bloom.AllocsPerOp,
			cmp.HashSet.AllocsPerOp,
			cmp.AllocRatio(),
		)
	}
	_ = tw.Flush()

	b.WriteString("\nHash strategy affects Bloom throughput only; sizing and hash set footprint are unchanged.\n")
	return b.String()
}

// FormatSizeSweep renders add-scenario comparisons across item counts.
func FormatSizeSweep(cfg Config, counts []uint64, results []Comparison) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Bloom filter vs hash set — size sweep (p=%.4f, add workload)\n\n",
		cfg.FalsePositiveRate)

	tw := tabwriter.NewWriter(&b, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "ITEMS n\tBLOOM B/item\tHASHSET B/item\tSPACE\tBLOOM ns/op\tHASHSET ns/op\tSPEEDUP\tBLOOM allocs/op\tHASHSET allocs/op\tALLOCS")
	for i, cmp := range results {
		fmt.Fprintf(tw, "%d\t%.1f\t%.1f\t%.1fx\t%.0f\t%.0f\t%.2fx\t%.2f\t%.2f\t%.1fx\n",
			counts[i],
			cmp.Bloom.BytesPerItem,
			cmp.HashSet.BytesPerItem,
			cmp.SpaceRatio(),
			cmp.Bloom.NsPerOp,
			cmp.HashSet.NsPerOp,
			cmp.SpeedRatio(),
			cmp.Bloom.AllocsPerOp,
			cmp.HashSet.AllocsPerOp,
			cmp.AllocRatio(),
		)
	}
	_ = tw.Flush()

	b.WriteString("\nBloom bytes/item stays near the theoretical minimum for fixed p; hash set footprint grows with key storage.\n")
	return b.String()
}

// FormatSizeSweepMarkdown renders the size sweep as a markdown table.
func FormatSizeSweepMarkdown(cfg Config, counts []uint64, results []Comparison) string {
	var b strings.Builder
	fmt.Fprintf(&b, "## Bloom filter vs hash set — size sweep\n\n")
	fmt.Fprintf(&b, "Add workload at `p=%.4f` across item counts.\n\n", cfg.FalsePositiveRate)
	fmt.Fprintln(&b, "| Items n | Bloom B/item | Hash set B/item | Space ratio | Bloom ns/op | Hash set ns/op | Speedup | Bloom allocs/op | Hash set allocs/op | Alloc ratio |")
	fmt.Fprintln(&b, "|---------|--------------|-----------------|-------------|-------------|----------------|---------|-----------------|--------------------|-------------|")
	for i, cmp := range results {
		fmt.Fprintf(&b, "| %d | %.1f | %.1f | %.1fx | %.0f | %.0f | %.2fx | %.2f | %.2f | %.1fx |\n",
			counts[i],
			cmp.Bloom.BytesPerItem,
			cmp.HashSet.BytesPerItem,
			cmp.SpaceRatio(),
			cmp.Bloom.NsPerOp,
			cmp.HashSet.NsPerOp,
			cmp.SpeedRatio(),
			cmp.Bloom.AllocsPerOp,
			cmp.HashSet.AllocsPerOp,
			cmp.AllocRatio(),
		)
	}
	return b.String()
}

// FormatHashSweepMarkdown renders the hash sweep as a markdown table.
func FormatHashSweepMarkdown(cfg Config, strategies []bloom.Strategy, results []Comparison) string {
	var b strings.Builder
	fmt.Fprintf(&b, "## Bloom filter vs hash set — hash sweep\n\n")
	fmt.Fprintf(&b, "Add workload at `n=%d`, `p=%.4f` across hash strategies.\n\n",
		cfg.ItemCount, cfg.FalsePositiveRate)
	fmt.Fprintln(&b, "| Hash | Bloom ns/op | Hash set ns/op | Speedup | Bloom B/item | Hash set B/item | Space ratio | Bloom allocs/op | Hash set allocs/op | Alloc ratio |")
	fmt.Fprintln(&b, "|------|-------------|----------------|---------|--------------|-----------------|-------------|-----------------|--------------------|-------------|")
	for i, cmp := range results {
		fmt.Fprintf(&b, "| %s | %.0f | %.0f | %.2fx | %.1f | %.1f | %.1fx | %.2f | %.2f | %.1fx |\n",
			strategies[i].String(),
			cmp.Bloom.NsPerOp,
			cmp.HashSet.NsPerOp,
			cmp.SpeedRatio(),
			cmp.Bloom.BytesPerItem,
			cmp.HashSet.BytesPerItem,
			cmp.SpaceRatio(),
			cmp.Bloom.AllocsPerOp,
			cmp.HashSet.AllocsPerOp,
			cmp.AllocRatio(),
		)
	}
	return b.String()
}

// FormatLookupMixSweep renders contains-mixed comparisons across hit ratios.
func FormatLookupMixSweep(cfg Config, ratios []float64, results []Comparison) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Bloom filter vs hash set — lookup mix sweep (n=%d, p=%.4f, lookup-repeats=%d)\n\n",
		cfg.ItemCount, cfg.FalsePositiveRate, cfg.LookupRepeats)

	tw := tabwriter.NewWriter(&b, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "HIT RATIO\tBLOOM ns/op\tHASHSET ns/op\tSPEEDUP\tBLOOM B/item\tHASHSET B/item\tSPACE\tBLOOM allocs/op\tHASHSET allocs/op\tALLOCS")
	for i, cmp := range results {
		fmt.Fprintf(tw, "%.0f%%\t%.0f\t%.0f\t%.2fx\t%.1f\t%.1f\t%.1fx\t%.2f\t%.2f\t%.1fx\n",
			ratios[i]*100,
			cmp.Bloom.NsPerOp,
			cmp.HashSet.NsPerOp,
			cmp.SpeedRatio(),
			cmp.Bloom.BytesPerItem,
			cmp.HashSet.BytesPerItem,
			cmp.SpaceRatio(),
			cmp.Bloom.AllocsPerOp,
			cmp.HashSet.AllocsPerOp,
			cmp.AllocRatio(),
		)
	}
	_ = tw.Flush()

	b.WriteString("\nHit ratio is the fraction of lookup keys present in the set. Bloom miss probes touch more unset bits; hash set miss probes exit after the first map slot.\n")
	return b.String()
}

// FormatLookupMixSweepMarkdown renders the lookup mix sweep as a markdown table.
func FormatLookupMixSweepMarkdown(cfg Config, ratios []float64, results []Comparison) string {
	var b strings.Builder
	fmt.Fprintf(&b, "## Bloom filter vs hash set — lookup mix sweep\n\n")
	fmt.Fprintf(&b, "Contains-mixed workload at `n=%d`, `p=%.4f`, lookup repeats `%d` across hit ratios.\n\n",
		cfg.ItemCount, cfg.FalsePositiveRate, cfg.LookupRepeats)
	fmt.Fprintln(&b, "| Hit ratio | Bloom ns/op | Hash set ns/op | Speedup | Bloom B/item | Hash set B/item | Space ratio | Bloom allocs/op | Hash set allocs/op | Alloc ratio |")
	fmt.Fprintln(&b, "|-----------|-------------|----------------|---------|--------------|-----------------|-------------|-----------------|--------------------|-------------|")
	for i, cmp := range results {
		fmt.Fprintf(&b, "| %.0f%% | %.0f | %.0f | %.2fx | %.1f | %.1f | %.1fx | %.2f | %.2f | %.1fx |\n",
			ratios[i]*100,
			cmp.Bloom.NsPerOp,
			cmp.HashSet.NsPerOp,
			cmp.SpeedRatio(),
			cmp.Bloom.BytesPerItem,
			cmp.HashSet.BytesPerItem,
			cmp.SpaceRatio(),
			cmp.Bloom.AllocsPerOp,
			cmp.HashSet.AllocsPerOp,
			cmp.AllocRatio(),
		)
	}
	return b.String()
}

func formatNotes(cmp Comparison) string {
	notes := ""
	if cmp.Bloom.TheoryFPR > 0 && cmp.Scenario != ScenarioMixedStream && cmp.Scenario != ScenarioContainsMixed {
		notes = fmt.Sprintf("theory FPR %.3f%%", cmp.Bloom.TheoryFPR*100)
	}
	if cmp.Scenario == ScenarioMixedStream {
		notes = fmt.Sprintf("dup calls bloom=%d hashset=%d", cmp.Bloom.FalsePositives, cmp.HashSet.FalsePositives)
	}
	if cmp.Scenario == ScenarioContainsMixed {
		notes = fmt.Sprintf("hit ratio %.0f%%", cmp.LookupHitRatio*100)
		if cmp.Bloom.TheoryFPR > 0 {
			notes += fmt.Sprintf("; theory FPR %.3f%%", cmp.Bloom.TheoryFPR*100)
		}
	}
	if cmp.Scenario == ScenarioRemove {
		if notes != "" {
			notes += "; "
		}
		notes += "counting bloom filter"
	}
	return notes
}
