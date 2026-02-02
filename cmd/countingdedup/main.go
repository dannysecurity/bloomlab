package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/dannysecurity/bloomlab/bloom"
	"github.com/dannysecurity/bloomlab/cmd/internal/filterflags"
	"github.com/dannysecurity/bloomlab/cmd/internal/streamflags"
	"github.com/dannysecurity/bloomlab/dedup"
)

func main() {
	flags := filterflags.RegisterCounting(100_000)
	stream := streamflags.Register()
	removePrefix := flag.String("remove-prefix", "-", "lines with this prefix remove the remainder from the set instead of classifying")
	flag.Parse()

	if err := stream.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "countingdedup: %v\n", err)
		os.Exit(2)
	}

	cc, err := flags.CountingConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "countingdedup: %v\n", err)
		os.Exit(1)
	}
	cf, err := bloom.NewCountingFilterFrom(cc)
	if err != nil {
		fmt.Fprintf(os.Stderr, "countingdedup: %v\n", err)
		os.Exit(1)
	}

	if flag.NArg() > 0 {
		fmt.Fprintln(os.Stderr, "countingdedup: reads lines from stdin; prefix lines with -remove-prefix to drop keys")
		fmt.Fprintln(os.Stderr, "Usage: countingdedup [flags] < lines.txt")
		os.Exit(2)
	}

	c := dedup.NewCountingClassifier(cf, stream.KeyFunc())
	if err := dedup.RunCounting(c, os.Stdin, dedup.CountingRunOptions{
		RunOptions:   stream.RunOptions(),
		RemovePrefix: *removePrefix,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "countingdedup: %v\n", err)
		os.Exit(1)
	}
}
