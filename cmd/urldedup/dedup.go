package main

import (
	"github.com/dannysecurity/bloomlab/bloom"
	"github.com/dannysecurity/bloomlab/cmd/internal/streamdedup"
	"github.com/dannysecurity/bloomlab/cmd/internal/urldedup"
)

type dedupOptions = urldedup.Options

// classify reports whether line is a duplicate in f and inserts it when novel.
// Empty lines are ignored and reported as not duplicate.
func classify(f *bloom.Filter, line string, opts dedupOptions) (duplicate bool, ok bool) {
	keyFn := func(line string) (string, bool) {
		return urldedup.Key(line, opts)
	}
	d := streamdedup.New(f, keyFn)
	return d.Classify(line)
}
