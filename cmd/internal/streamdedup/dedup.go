package streamdedup

import (
	"strings"

	"github.com/dannysecurity/bloomlab/bloom"
)

// KeyFunc maps a line to a dedup key. Return ok=false to skip the line.
type KeyFunc func(line string) (key string, ok bool)

// TrimKey trims whitespace and skips blank lines.
func TrimKey(line string) (string, bool) {
	line = strings.TrimSpace(line)
	if line == "" {
		return "", false
	}
	return line, true
}

// IgnoreCaseKey trims whitespace, lowercases, and skips blank lines.
func IgnoreCaseKey(line string) (string, bool) {
	key, ok := TrimKey(line)
	if !ok {
		return "", false
	}
	return strings.ToLower(key), true
}

// Deduper classifies stdin lines as novel or duplicate using a Bloom filter.
type Deduper struct {
	filter *bloom.Filter
	keyFn  KeyFunc
	novel  int
	dup    int
}

// New builds a Deduper. keyFn defaults to TrimKey when nil.
func New(f *bloom.Filter, keyFn KeyFunc) *Deduper {
	if keyFn == nil {
		keyFn = TrimKey
	}
	return &Deduper{filter: f, keyFn: keyFn}
}

// Classify reports whether line is a duplicate and inserts it when novel.
func (d *Deduper) Classify(line string) (duplicate bool, ok bool) {
	key, ok := d.keyFn(line)
	if !ok {
		return false, false
	}
	keyBytes := []byte(key)
	if d.filter.Contains(keyBytes) {
		d.dup++
		return true, true
	}
	d.filter.Add(keyBytes)
	d.novel++
	return false, true
}

// Filter returns the underlying Bloom filter.
func (d *Deduper) Filter() *bloom.Filter { return d.filter }

// Novel returns the number of first-seen lines.
func (d *Deduper) Novel() int { return d.novel }

// Duplicates returns the number of duplicate lines seen.
func (d *Deduper) Duplicates() int { return d.dup }
