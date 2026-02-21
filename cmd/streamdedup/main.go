package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/dannysecurity/bloomlab/bloom"
	"github.com/dannysecurity/bloomlab/cmd/internal/filterflags"
	"github.com/dannysecurity/bloomlab/cmd/internal/streamdedup"
	"github.com/dannysecurity/bloomlab/cmd/internal/streamflags"
	"github.com/dannysecurity/bloomlab/dedup"
)

func main() {
	flags := filterflags.Register(100_000)
	stream := streamflags.Register()
	flag.Parse()

	if err := stream.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "streamdedup: %v\n", err)
		os.Exit(2)
	}

	fc, err := flags.FilterConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "streamdedup: %v\n", err)
		os.Exit(1)
	}
	f, err := bloom.NewFilterFrom(fc)
	if err != nil {
		fmt.Fprintf(os.Stderr, "streamdedup: %v\n", err)
		os.Exit(1)
	}

	src, _, closeFn, err := dedup.OpenInput(dedup.InputModeFile, flag.Args())
	if err != nil {
		fmt.Fprintf(os.Stderr, "streamdedup: %v\n", err)
		os.Exit(2)
	}
	defer closeFn()

	d := streamdedup.New(f, stream.KeyFunc())
	if err := streamdedup.Run(d, src.Reader, streamdedup.RunOptions(stream.RunOptions())); err != nil {
		fmt.Fprintf(os.Stderr, "streamdedup: %v\n", err)
		os.Exit(1)
	}
}
