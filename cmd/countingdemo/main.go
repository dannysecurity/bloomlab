package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/dannysecurity/bloomlab/bloom"
	"github.com/dannysecurity/bloomlab/cmd/internal/filterflags"
)

func main() {
	flags := filterflags.RegisterCounting(10_000)
	remove := flag.Bool("remove", false, "remove words instead of adding")
	flag.Parse()

	cfg, err := flags.Config()
	if err != nil {
		fmt.Fprintf(os.Stderr, "countingdemo: %v\n", err)
		os.Exit(1)
	}
	cf, err := bloom.NewCountingFilter(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "countingdemo: %v\n", err)
		os.Exit(1)
	}

	args := flag.Args()
	if len(args) == 0 {
		fmt.Println(cfg.String())
		fmt.Println("Usage: countingdemo [flags] <word> ...")
		os.Exit(0)
	}

	for _, word := range args {
		key := []byte(word)
		if *remove {
			if cf.Remove(key) {
				fmt.Printf("%q: removed\n", word)
			} else {
				fmt.Printf("%q: not present (remove skipped)\n", word)
			}
			continue
		}

		if cf.Contains(key) {
			fmt.Printf("%q: maybe present\n", word)
		} else if err := cf.Add(key); err != nil {
			fmt.Fprintf(os.Stderr, "add %q: %v\n", word, err)
		} else {
			fmt.Printf("%q: added\n", word)
		}
	}

	fmt.Printf("fill ratio: %.2f%%, inserts: %d, theoretical FPR: %.4f%%\n",
		cf.FillRatio()*100, cf.ApproximateCount(), cf.TheoryFPR()*100)
	if !*remove && len(args) > 0 {
		fmt.Printf("checked: %s\n", strings.Join(args, ", "))
	}
}
