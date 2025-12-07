package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"

	"github.com/dannysecurity/bloomlab/bloom"
	"github.com/dannysecurity/bloomlab/cmd/internal/filterflags"
)

func main() {
	flags := filterflags.Register(100_000)
	quiet := flag.Bool("quiet", false, "print summary only")
	flag.Parse()

	cfg, err := flags.Config()
	if err != nil {
		fmt.Fprintf(os.Stderr, "urldedup: %v\n", err)
		os.Exit(1)
	}
	f, err := bloom.NewFilter(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "urldedup: %v\n", err)
		os.Exit(1)
	}

	if flag.NArg() > 0 {
		fmt.Fprintln(os.Stderr, "urldedup: reads lines from stdin; flags configure the Bloom filter")
		fmt.Fprintln(os.Stderr, "Usage: urldedup [flags] < urls.txt")
		os.Exit(2)
	}

	scanner := bufio.NewScanner(os.Stdin)
	var novel, dup int
	for scanner.Scan() {
		line := scanner.Text()
		isDup, ok := classify(f, line)
		if !ok {
			continue
		}
		if isDup {
			dup++
			if !*quiet {
				fmt.Printf("dup\t%s\n", line)
			}
			continue
		}
		novel++
		if !*quiet {
			fmt.Printf("new\t%s\n", line)
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "urldedup: read stdin: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "novel: %d, duplicates: %d, inserts: %d, fill: %.2f%%, theory FPR: %.4f%%\n",
		novel, dup, f.ApproximateCount(), f.FillRatio()*100, f.TheoryFPR()*100)
}
