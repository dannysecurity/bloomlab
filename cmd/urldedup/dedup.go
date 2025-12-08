package main

import (
	"github.com/dannysecurity/bloomlab/bloom"
)

type dedupOptions struct {
	normalize bool
}

// classify reports whether line is a duplicate in f and inserts it when novel.
// Empty lines are ignored and reported as not duplicate.
func classify(f *bloom.Filter, line string, opts dedupOptions) (duplicate bool, ok bool) {
	key, ok := canonicalKey(line, opts.normalize)
	if !ok {
		return false, false
	}
	keyBytes := []byte(key)
	if f.Contains(keyBytes) {
		return true, true
	}
	f.Add(keyBytes)
	return false, true
}
