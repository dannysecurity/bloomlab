package main

import (
	"github.com/dannysecurity/bloomlab/bloom"
	"github.com/dannysecurity/bloomlab/cmd/internal/streamdedup"
)

type dedupOptions struct {
	normalize bool
}

// classify reports whether line is a duplicate in f and inserts it when novel.
// Empty lines are ignored and reported as not duplicate.
func classify(f *bloom.Filter, line string, opts dedupOptions) (duplicate bool, ok bool) {
	keyFn := func(line string) (string, bool) {
		return canonicalKey(line, opts.normalize)
	}
	d := streamdedup.New(f, keyFn)
	return d.Classify(line)
}
