package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/dannysecurity/bloomlab/benchcompare"
)

func main() {
	n := flag.Uint64("n", 100_000, "expected item count for Bloom filter sizing")
	p := flag.Float64("p", 0.01, "target false positive rate (0, 1)")
	repeats := flag.Int("repeats", 4, "lookup repetitions for contains scenarios")
	markdown := flag.Bool("markdown", false, "emit markdown table instead of plain text")
	flag.Parse()

	cfg := benchcompare.Config{
		ItemCount:         *n,
		FalsePositiveRate: *p,
		LookupRepeats:     *repeats,
	}

	results, err := benchcompare.Compare(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "benchcompare: %v\n", err)
		os.Exit(1)
	}

	if *markdown {
		fmt.Print(benchcompare.FormatMarkdown(cfg, results))
		return
	}
	fmt.Print(benchcompare.FormatReport(cfg, results))
}
