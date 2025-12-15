package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/dannysecurity/bloomlab/bloom"
)

func main() {
	n := flag.Uint64("n", 10_000, "expected number of distinct inserts")
	p := flag.Float64("p", 0.01, "target false positive rate in (0, 1)")
	at := flag.Uint64("at", 0, "evaluate theory FPR at this insert count (default: n)")
	flag.Parse()

	plan, err := bloom.PlanSizing(*n, *p)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fprcalc: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(plan)

	inserts := *at
	if inserts == 0 {
		inserts = *n
	}
	if inserts != *n {
		fpr := bloom.TheoryFalsePositiveRate(inserts, plan.Bits, plan.HashCount)
		fill := bloom.TheoryFillFraction(inserts, plan.Bits, plan.HashCount)
		fmt.Printf(
			"at n=%d: fill≈%.3f (%.1f%%), theory FPR≈%.5f (%.3f%%)\n",
			inserts, fill, fill*100, fpr, fpr*100,
		)
	}
}
