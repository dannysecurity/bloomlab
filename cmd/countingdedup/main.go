package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/dannysecurity/bloomlab/bloom"
	"github.com/dannysecurity/bloomlab/cmd/internal/filterflags"
	"github.com/dannysecurity/bloomlab/dedup"
)

func main() {
	flags := filterflags.RegisterCounting(100_000)
	quiet := flag.Bool("quiet", false, "print summary only")
	novelOnly := flag.Bool("novel-only", false, "emit first-seen lines only")
	ignoreCase := flag.Bool("ignore-case", false, "compare lines case-insensitively")
	removePrefix := flag.String("remove-prefix", "-", "lines with this prefix remove the remainder from the set instead of classifying")
	jsonOut := flag.Bool("json", false, "emit one JSON object per line on stdout")
	flag.Parse()

	cfg, err := flags.Config()
	if err != nil {
		fmt.Fprintf(os.Stderr, "countingdedup: %v\n", err)
		os.Exit(1)
	}
	cf, err := bloom.NewCountingFilter(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "countingdedup: %v\n", err)
		os.Exit(1)
	}

	if flag.NArg() > 0 {
		fmt.Fprintln(os.Stderr, "countingdedup: reads lines from stdin; prefix lines with -remove-prefix to drop keys")
		fmt.Fprintln(os.Stderr, "Usage: countingdedup [flags] < lines.txt")
		os.Exit(2)
	}

	keyFn := dedup.TrimKey
	if *ignoreCase {
		keyFn = dedup.IgnoreCaseKey
	}

	format := dedup.FormatText
	if *jsonOut {
		format = dedup.FormatJSON
	}

	c := dedup.NewCountingClassifier(cf, keyFn)
	if err := dedup.RunCounting(c, os.Stdin, dedup.CountingRunOptions{
		RunOptions: dedup.RunOptions{
			Quiet:     *quiet,
			NovelOnly: *novelOnly,
			Format:    format,
		},
		RemovePrefix: *removePrefix,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "countingdedup: %v\n", err)
		os.Exit(1)
	}
}
