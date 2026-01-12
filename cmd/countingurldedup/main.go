package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/dannysecurity/bloomlab/bloom"
	"github.com/dannysecurity/bloomlab/cmd/internal/filterflags"
	"github.com/dannysecurity/bloomlab/cmd/internal/streamflags"
	"github.com/dannysecurity/bloomlab/cmd/internal/urlflags"
	"github.com/dannysecurity/bloomlab/dedup"
)

func main() {
	flags := filterflags.RegisterCounting(100_000)
	stream := streamflags.Register()
	url := urlflags.Register()
	removePrefix := flag.String("remove-prefix", "-", "lines with this prefix remove the remainder from the set instead of classifying")
	flag.Parse()

	cfg, err := flags.Config()
	if err != nil {
		fmt.Fprintf(os.Stderr, "countingurldedup: %v\n", err)
		os.Exit(1)
	}
	cf, err := bloom.NewCountingFilter(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "countingurldedup: %v\n", err)
		os.Exit(1)
	}

	if flag.NArg() > 0 {
		fmt.Fprintln(os.Stderr, "countingurldedup: reads lines from stdin; prefix lines with -remove-prefix to drop keys")
		fmt.Fprintln(os.Stderr, "Usage: countingurldedup [flags] < urls.txt")
		os.Exit(2)
	}

	c := dedup.NewCountingClassifier(cf, url.KeyFunc())
	if err := dedup.RunCounting(c, os.Stdin, dedup.CountingRunOptions{
		RunOptions:   stream.RunOptions(),
		RemovePrefix: *removePrefix,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "countingurldedup: %v\n", err)
		os.Exit(1)
	}
}
