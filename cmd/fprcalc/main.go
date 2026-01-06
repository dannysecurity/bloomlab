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
	at := flag.Uint64("at", 0, "evaluate theory FPR at this insert count (default: n)")
	derive := flag.Bool("derive", false, "print step-by-step false-positive math")
	flag.Parse()

	bloomCfg, err := filter.Config()
	if err != nil {
		fmt.Fprintf(os.Stderr, "fprcalc: %v\n", err)
		os.Exit(1)
	}

	plan, err := bloom.PlanSizing(bloomCfg.ExpectedCapacity, bloomCfg.FalsePositiveRate)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fprcalc: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(plan)
	if *derive {
		fmt.Println()
		fmt.Print(bloom.FormatSizingDerivation(plan))
	}

	inserts := *at
	if inserts == 0 {
		inserts = bloomCfg.ExpectedCapacity
	}
	if inserts != bloomCfg.ExpectedCapacity {
		fpr := bloom.TheoryFalsePositiveRate(inserts, plan.Bits, plan.HashCount)
		fill := bloom.TheoryFillFraction(inserts, plan.Bits, plan.HashCount)
		fmt.Printf(
			"at n=%d: fill≈%.3f (%.1f%%), theory FPR≈%.5f (%.3f%%)\n",
			inserts, fill, fill*100, fpr, fpr*100,
		)
	}
}
