package main

import (
	"strings"

	"github.com/dannysecurity/bloomlab/bloom"
)

// classify reports whether line is a duplicate in f and inserts it when novel.
// Empty lines are ignored and reported as not duplicate.
func classify(f *bloom.Filter, line string) (duplicate bool, ok bool) {
	key := []byte(strings.TrimSpace(line))
	if len(key) == 0 {
		return false, false
	}
	if f.Contains(key) {
		return true, true
	}
	f.Add(key)
	return false, true
}
