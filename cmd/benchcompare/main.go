package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/dannysecurity/bloomlab/benchcompare"
	"github.com/dannysecurity/bloomlab/cmd/internal/filterflags"
)

func main() {
	filter := filterflags.Register(100_000)
	repeats := flag.Int("repeats", 4, "lookup repetitions for contains scenarios")
	sweepFPR := flag.Bool("sweep-fpr", false, "compare add workload across FPR targets instead of all scenarios")
	sweepHash := flag.Bool("sweep-hash", false, "compare add workload across hash strategies instead of all scenarios")
	sweepSize := flag.Bool("sweep-size", false, "compare add workload across item counts instead of all scenarios")
	sweepMix := flag.Bool("sweep-mix", false, "compare mixed lookup workload across hit ratios instead of all scenarios")
	pValues := flag.String("p-values", "0.001,0.01,0.1", "comma-separated FPR targets for -sweep-fpr")
	hashValues := flag.String("hash-values", "fnv,murmur3,xxhash,wyhash", "comma-separated hash strategies for -sweep-hash")
	sizeValues := flag.String("size-values", "10000,50000,100000,500000", "comma-separated item counts for -sweep-size")
	mixValues := flag.String("mix-values", "0,0.25,0.5,0.75,1", "comma-separated lookup hit ratios for -sweep-mix")
	hitRatio := flag.Float64("hit-ratio", 0.5, "fraction of lookup keys present for contains_mixed scenario")
	markdown := flag.Bool("markdown", false, "emit markdown table instead of plain text")
	flag.Parse()

	fc, err := filter.FilterConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "benchcompare: %v\n", err)
		os.Exit(1)
	}

	cfg := benchcompare.NewConfig(fc)
	cfg.LookupRepeats = *repeats
	cfg.LookupHitRatio = *hitRatio

	if *sweepMix {
		ratios, err := benchcompare.ParseLookupMixRatios(*mixValues)
		if err != nil {
			fmt.Fprintf(os.Stderr, "benchcompare: %v\n", err)
			os.Exit(1)
		}
		results, err := benchcompare.CompareLookupMixSweep(cfg, ratios)
		if err != nil {
			fmt.Fprintf(os.Stderr, "benchcompare: %v\n", err)
			os.Exit(1)
		}
		if *markdown {
			fmt.Print(benchcompare.FormatLookupMixSweepMarkdown(cfg, ratios, results))
			return
		}
		fmt.Print(benchcompare.FormatLookupMixSweep(cfg, ratios, results))
		return
	}

	if *sweepFPR {
		rates, err := benchcompare.ParseFPRRates(*pValues)
		if err != nil {
			fmt.Fprintf(os.Stderr, "benchcompare: %v\n", err)
			os.Exit(1)
		}
		results, err := benchcompare.CompareFPRSweep(cfg, rates)
		if err != nil {
			fmt.Fprintf(os.Stderr, "benchcompare: %v\n", err)
			os.Exit(1)
		}
		if *markdown {
			fmt.Print(benchcompare.FormatFPRSweepMarkdown(cfg, rates, results))
			return
		}
		fmt.Print(benchcompare.FormatFPRSweep(cfg, rates, results))
		return
	}

	if *sweepHash {
		strategies, err := benchcompare.ParseHashStrategies(*hashValues)
		if err != nil {
			fmt.Fprintf(os.Stderr, "benchcompare: %v\n", err)
			os.Exit(1)
		}
		results, err := benchcompare.CompareHashSweep(cfg, strategies)
		if err != nil {
			fmt.Fprintf(os.Stderr, "benchcompare: %v\n", err)
			os.Exit(1)
		}
		if *markdown {
			fmt.Print(benchcompare.FormatHashSweepMarkdown(cfg, strategies, results))
			return
		}
		fmt.Print(benchcompare.FormatHashSweep(cfg, strategies, results))
		return
	}

	if *sweepSize {
		counts, err := benchcompare.ParseSizeCounts(*sizeValues)
		if err != nil {
			fmt.Fprintf(os.Stderr, "benchcompare: %v\n", err)
			os.Exit(1)
		}
		results, err := benchcompare.CompareSizeSweep(cfg, counts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "benchcompare: %v\n", err)
			os.Exit(1)
		}
		if *markdown {
			fmt.Print(benchcompare.FormatSizeSweepMarkdown(cfg, counts, results))
			return
		}
		fmt.Print(benchcompare.FormatSizeSweep(cfg, counts, results))
		return
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
