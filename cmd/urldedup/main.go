package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/dannysecurity/bloomlab/bloom"
	"github.com/dannysecurity/bloomlab/cmd/internal/filterflags"
	"github.com/dannysecurity/bloomlab/cmd/internal/streamdedup"
	"github.com/dannysecurity/bloomlab/cmd/internal/streamflags"
	"github.com/dannysecurity/bloomlab/cmd/internal/urlflags"
)

func main() {
	flags := filterflags.Register(100_000)
	stream := streamflags.Register()
	url := urlflags.Register()
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

	d := streamdedup.New(f, url.KeyFunc())
	if err := streamdedup.Run(d, os.Stdin, streamdedup.RunOptions(stream.RunOptions())); err != nil {
		fmt.Fprintf(os.Stderr, "urldedup: %v\n", err)
		os.Exit(1)
	}
}
