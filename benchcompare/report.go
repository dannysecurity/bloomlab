package benchcompare

import (
	"fmt"
	"strings"
	"text/tabwriter"
)

// FormatReport renders comparisons as a fixed-width table suitable for stdout.
func FormatReport(cfg Config, results []Comparison) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Bloom filter vs hash set (n=%d, p=%.4f, lookup-repeats=%d)\n\n",
		cfg.ItemCount, cfg.FalsePositiveRate, cfg.LookupRepeats)

	tw := tabwriter.NewWriter(&b, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "SCENARIO\tBLOOM ns/op\tHASHSET ns/op\tSPEEDUP\tBLOOM B/item\tHASHSET B/item\tSPACE\tNOTES")
	for _, cmp := range results {
		notes := ""
		if cmp.Bloom.TheoryFPR > 0 {
			notes = fmt.Sprintf("theory FPR %.3f%%", cmp.Bloom.TheoryFPR*100)
		}
		if cmp.Scenario == ScenarioMixedStream {
			notes = fmt.Sprintf("dup calls bloom=%d hashset=%d", cmp.Bloom.FalsePositives, cmp.HashSet.FalsePositives)
		}
		fmt.Fprintf(tw, "%s\t%.0f\t%.0f\t%.2fx\t%.1f\t%.1f\t%.1fx\t%s\n",
			cmp.Scenario,
			cmp.Bloom.NsPerOp,
			cmp.HashSet.NsPerOp,
			cmp.SpeedRatio(),
			cmp.Bloom.BytesPerItem,
			cmp.HashSet.BytesPerItem,
			cmp.SpaceRatio(),
			notes,
		)
	}
	_ = tw.Flush()

	b.WriteString("\nSpeedup > 1 means Bloom was faster; space > 1 means Bloom used less memory per item.\n")
	return b.String()
}

// FormatMarkdown renders comparisons as a GitHub-flavored markdown table.
func FormatMarkdown(cfg Config, results []Comparison) string {
	var b strings.Builder
	fmt.Fprintf(&b, "## Bloom filter vs hash set\n\n")
	fmt.Fprintf(&b, "Configuration: `n=%d`, `p=%.4f`, lookup repeats `%d`.\n\n",
		cfg.ItemCount, cfg.FalsePositiveRate, cfg.LookupRepeats)
	fmt.Fprintln(&b, "| Scenario | Bloom ns/op | Hash set ns/op | Speedup | Bloom B/item | Hash set B/item | Space ratio | Notes |")
	fmt.Fprintln(&b, "|----------|-------------|----------------|---------|--------------|-----------------|-------------|-------|")
	for _, cmp := range results {
		notes := ""
		if cmp.Bloom.TheoryFPR > 0 && cmp.Scenario != ScenarioMixedStream {
			notes = fmt.Sprintf("theory FPR %.3f%%", cmp.Bloom.TheoryFPR*100)
		}
		if cmp.Scenario == ScenarioMixedStream {
			notes = fmt.Sprintf("dup calls bloom=%d hashset=%d", cmp.Bloom.FalsePositives, cmp.HashSet.FalsePositives)
		}
		fmt.Fprintf(&b, "| %s | %.0f | %.0f | %.2fx | %.1f | %.1f | %.1fx | %s |\n",
			cmp.Scenario,
			cmp.Bloom.NsPerOp,
			cmp.HashSet.NsPerOp,
			cmp.SpeedRatio(),
			cmp.Bloom.BytesPerItem,
			cmp.HashSet.BytesPerItem,
			cmp.SpaceRatio(),
			notes,
		)
	}
	return b.String()
}
