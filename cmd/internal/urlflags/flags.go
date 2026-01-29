package urlflags

import (
	"flag"

	"github.com/dannysecurity/bloomlab/cmd/internal/urldedup"
	"github.com/dannysecurity/bloomlab/dedup"
)

// Flags holds shared CLI options for URL stream dedup tools.
type Flags struct {
	Normalize     *bool
	StripQuery    *bool
	StripTracking *bool
	StripFragment *bool
	DomainOnly    *bool
}

// Register binds URL canonicalization flags shared by urldedup and countingurldedup.
func Register() *Flags {
	return &Flags{
		Normalize:     flag.Bool("normalize", false, "canonicalize URLs (scheme/host case, default ports, trailing slashes, fragments)"),
		StripQuery:    flag.Bool("strip-query", false, "ignore query strings when deduplicating"),
		StripTracking: flag.Bool("strip-tracking", false, "drop common marketing/click-tracking query parameters (utm_*, fbclid, gclid, etc.)"),
		StripFragment: flag.Bool("strip-fragment", false, "ignore URL fragments (#...) when deduplicating"),
		DomainOnly:    flag.Bool("domain-only", false, "deduplicate by host name only, ignoring path and query"),
	}
}

// Options builds urldedup.Options from parsed flag values.
func (f *Flags) Options() urldedup.Options {
	return urldedup.Options{
		Normalize:     *f.Normalize,
		StripQuery:    *f.StripQuery,
		StripTracking: *f.StripTracking,
		StripFragment: *f.StripFragment,
		DomainOnly:    *f.DomainOnly,
	}
}

// KeyFunc returns the URL key function for dedup classifiers.
func (f *Flags) KeyFunc() dedup.KeyFunc {
	opts := f.Options()
	return func(line string) (string, bool) {
		return urldedup.Key(line, opts)
	}
}
