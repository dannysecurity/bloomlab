package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/dannysecurity/bloomlab/bloom"
	"github.com/dannysecurity/bloomlab/cmd/internal/filterflags"
)

func main() {
	filter := filterflags.Register(10_000)
	samples := flag.Int("samples", 10_000, "synthetic keys to probe for spread measurement")
	keyPrefix := flag.String("key-prefix", "hashtune", "prefix for synthetic probe keys")
	seedsRaw := flag.String("seeds", "", "comma-separated candidate seeds (default: built-in ladder)")
	hashValues := flag.String("hash-values", "", "comma-separated hash strategies (default: all)")
	strategyOnly := flag.String("strategy", "", "tune seeds for one strategy only (skip cross-strategy comparison)")
	markdown := flag.Bool("markdown", false, "emit markdown table instead of plain text")
	flag.Parse()

	bloomCfg, err := filter.Config()
	if err != nil {
		fmt.Fprintf(os.Stderr, "hashtune: %v\n", err)
		os.Exit(1)
	}

	opts, err := bloom.TuneOptionsFromConfig(bloomCfg, *samples, *keyPrefix)
	if err != nil {
		fmt.Fprintf(os.Stderr, "hashtune: %v\n", err)
		os.Exit(1)
	}

	seeds, err := bloom.ParseSeeds(*seedsRaw)
	if err != nil {
		fmt.Fprintf(os.Stderr, "hashtune: %v\n", err)
		os.Exit(1)
	}
	if len(seeds) == 0 {
		seeds = bloom.DefaultTuneSeeds()
	}

	if *strategyOnly != "" {
		strategy, err := bloom.ParseStrategy(*strategyOnly)
		if err != nil {
			fmt.Fprintf(os.Stderr, "hashtune: %v\n", err)
			os.Exit(1)
		}
		report := bloom.TuningReport{Options: opts}
		report.Candidates = bloom.CompareSeeds(strategy, opts, seeds)
		if len(report.Candidates) > 0 {
			best := report.Candidates[0]
			report.Best = bloom.StrategyScore{
				Strategy: strategy,
				Seed:     best.Seed,
				Spread:   best.Spread,
			}
		}
		printReport(report, *markdown)
		return
	}

	strategies := bloom.AllStrategies()
	if *hashValues != "" {
		strategies, err = bloom.ParseStrategyList(*hashValues)
		if err != nil {
			fmt.Fprintf(os.Stderr, "hashtune: %v\n", err)
			os.Exit(1)
		}
	}

	report, err := bloom.RecommendHasherFromConfig(bloomCfg, *samples, *keyPrefix, strategies, seeds)
	if err != nil {
		fmt.Fprintf(os.Stderr, "hashtune: %v\n", err)
		os.Exit(1)
	}
	printReport(report, *markdown)
}

func printReport(report bloom.TuningReport, markdown bool) {
	if markdown {
		fmt.Print(bloom.FormatTuningReportMarkdown(report))
		return
	}
	fmt.Print(bloom.FormatTuningReport(report))
}
