package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/dannysecurity/bloomlab/bloom"
	"github.com/dannysecurity/bloomlab/cmd/internal/filterflags"
	"github.com/dannysecurity/bloomlab/cmd/internal/streamdedup"
)

func main() {
	flags := filterflags.Register(100_000)
	quiet := flag.Bool("quiet", false, "print summary only")
	ignoreCase := flag.Bool("ignore-case", false, "compare lines case-insensitively")
	jsonOut := flag.Bool("json", false, "emit one JSON object per line on stdout")
	flag.Parse()

	cfg, err := flags.Config()
	if err != nil {
		fmt.Fprintf(os.Stderr, "streamdedup: %v\n", err)
		os.Exit(1)
	}
	f, err := bloom.NewFilter(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "streamdedup: %v\n", err)
		os.Exit(1)
	}

	if flag.NArg() > 0 {
		fmt.Fprintln(os.Stderr, "streamdedup: reads lines from stdin; flags configure the Bloom filter")
		fmt.Fprintln(os.Stderr, "Usage: streamdedup [flags] < lines.txt")
		os.Exit(2)
	}

	keyFn := streamdedup.TrimKey
	if *ignoreCase {
		keyFn = streamdedup.IgnoreCaseKey
	}

	format := streamdedup.FormatText
	if *jsonOut {
		format = streamdedup.FormatJSON
	}

	d := streamdedup.New(f, keyFn)
	if err := streamdedup.Run(d, os.Stdin, streamdedup.RunOptions{
		Quiet:  *quiet,
		Format: format,
	}); err != nil {
		fmt.Fprintf(os.Stderr, "streamdedup: %v\n", err)
		os.Exit(1)
	}
}
