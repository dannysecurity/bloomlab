package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/dannysecurity/bloomlab/benchcompare"
	"github.com/dannysecurity/bloomlab/bloom"
)

func main() {
	n := flag.Uint64("n", 100_000, "expected item count for Bloom filter sizing")
	p := flag.Float64("p", 0.01, "target false positive rate (0, 1)")
	repeats := flag.Int("repeats", 4, "lookup repetitions for contains scenarios")
	hashName := flag.String("hash", "fnv", "Bloom hash strategy: fnv, murmur3, xxhash")
	seed := flag.Uint64("seed", 0, "Bloom hash seed")
	sweepFPR := flag.Bool("sweep-fpr", false, "compare add workload across FPR targets instead of all scenarios")
	sweepHash := flag.Bool("sweep-hash", false, "compare add workload across hash strategies instead of all scenarios")
	pValues := flag.String("p-values", "0.001,0.01,0.1", "comma-separated FPR targets for -sweep-fpr")
	hashValues := flag.String("hash-values", "fnv,murmur3,xxhash", "comma-separated hash strategies for -sweep-hash")
	markdown := flag.Bool("markdown", false, "emit markdown table instead of plain text")
	flag.Parse()

	strategy, err := bloom.ParseStrategy(*hashName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "benchcompare: %v\n", err)
		os.Exit(1)
	}

	cfg := benchcompare.Config{
		ItemCount:         *n,
		FalsePositiveRate: *p,
		LookupRepeats:     *repeats,
		Hash: bloom.HashConfig{
			Strategy: strategy,
			Seed:     *seed,
		},
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
