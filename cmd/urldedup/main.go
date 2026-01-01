package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/dannysecurity/bloomlab/bloom"
	"github.com/dannysecurity/bloomlab/cmd/internal/filterflags"
	"github.com/dannysecurity/bloomlab/cmd/internal/streamdedup"
	"github.com/dannysecurity/bloomlab/cmd/internal/urldedup"
)

func main() {
	flags := filterflags.Register(100_000)
	quiet := flag.Bool("quiet", false, "print summary only")
	normalize := flag.Bool("normalize", false, "canonicalize URLs (scheme/host case, default ports, trailing slashes, fragments)")
	stripQuery := flag.Bool("strip-query", false, "ignore query strings when deduplicating")
	stripTracking := flag.Bool("strip-tracking", false, "drop common marketing/click-tracking query parameters (utm_*, fbclid, gclid, etc.)")
	domainOnly := flag.Bool("domain-only", false, "deduplicate by host name only, ignoring path and query")
	jsonOut := flag.Bool("json", false, "emit one JSON object per line on stdout")
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

	opts := urldedup.Options{
		Normalize:     *normalize,
		StripQuery:    *stripQuery,
		StripTracking: *stripTracking,
		DomainOnly:    *domainOnly,
	}
	keyFn := func(line string) (string, bool) {
		return urldedup.Key(line, opts)
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
		fmt.Fprintf(os.Stderr, "urldedup: %v\n", err)
		os.Exit(1)
	}
}
