package main

import (
	"github.com/dannysecurity/bloomlab/bloom"
	"github.com/dannysecurity/bloomlab/cmd/internal/urldedup"
	"github.com/dannysecurity/bloomlab/dedup"
)

type dedupOptions = urldedup.Options

func keyFn(opts dedupOptions) dedup.KeyFunc {
	return func(line string) (string, bool) {
		return urldedup.Key(line, opts)
	}
}

// classify reports whether line is a duplicate in cf and inserts it when novel.
func classify(cf *bloom.CountingFilter, line string, opts dedupOptions) (duplicate bool, ok bool, err error) {
	c := dedup.NewCountingClassifier(cf, keyFn(opts))
	return c.Classify(line)
}

// remove drops a previously inserted key from cf when present.
func remove(cf *bloom.CountingFilter, line string, opts dedupOptions) (removed bool, ok bool) {
	c := dedup.NewCountingClassifier(cf, keyFn(opts))
	return c.Remove(line)
}
