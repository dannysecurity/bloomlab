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
	"github.com/dannysecurity/bloomlab/dedup"
)

func main() {
	flags := filterflags.Register(10_000)
	stream := streamflags.Register()
	url := urlflags.Register()
	urlMode := flag.Bool("url", false, "treat input lines as URLs (enables -normalize, -strip-query, etc.)")
	sampleFlag := flag.String("sample", "", "built-in demo dataset when no input: stream, url, or tracking")
	flag.Parse()

	if err := stream.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "dedupdemo: %v\n", err)
		os.Exit(2)
	}

	fc, err := flags.FilterConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "dedupdemo: %v\n", err)
		os.Exit(1)
	}
	f, err := bloom.NewFilterFrom(fc)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dedupdemo: %v\n", err)
		os.Exit(1)
	}

	src, sampleNeeded, closeFn, err := dedup.OpenInput(dedup.InputModeLines, flag.Args())
	if err != nil {
		fmt.Fprintf(os.Stderr, "dedupdemo: %v\n", err)
		os.Exit(1)
	}
	defer closeFn()

	if sampleNeeded {
		kind := dedup.DefaultSampleKind(*urlMode)
		if *sampleFlag != "" {
			kind, err = dedup.ParseSampleKind(*sampleFlag)
			if err != nil {
				fmt.Fprintf(os.Stderr, "dedupdemo: %v\n", err)
				os.Exit(2)
			}
		}
		reader, err := dedup.SampleReader(kind)
		if err != nil {
			fmt.Fprintf(os.Stderr, "dedupdemo: %v\n", err)
			os.Exit(2)
		}
		src = dedup.InputSource{Reader: reader, Label: "sample:" + string(kind)}
		fmt.Fprintf(os.Stderr, "dedupdemo: running %s sample (%s); pipe stdin or pass lines as args\n",
			kind, fc.String())
	}

	keyFn := stream.KeyFunc()
	if *urlMode {
		keyFn = url.KeyFunc()
	}

	d := streamdedup.New(f, keyFn)
	if err := streamdedup.Run(d, src.Reader, streamdedup.RunOptions(stream.RunOptions())); err != nil {
		fmt.Fprintf(os.Stderr, "dedupdemo: %v\n", err)
		os.Exit(1)
	}
}
