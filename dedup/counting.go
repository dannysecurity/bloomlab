package dedup

import "github.com/dannysecurity/bloomlab/bloom"

// CountingClassifier supports check-then-insert dedup with explicit removals
// via a counting Bloom filter.
type CountingClassifier struct {
	filter  *bloom.CountingFilter
	keyFn   KeyFunc
	novel   int
	dup     int
	removed int
}

// NewCountingClassifier builds a CountingClassifier. keyFn defaults to TrimKey when nil.
func NewCountingClassifier(f *bloom.CountingFilter, keyFn KeyFunc) *CountingClassifier {
	if keyFn == nil {
		keyFn = TrimKey
	}
	return &CountingClassifier{filter: f, keyFn: keyFn}
}

// Classify reports whether line is a duplicate and inserts it when novel.
func (c *CountingClassifier) Classify(line string) (duplicate bool, ok bool, err error) {
	key, ok := c.keyFn(line)
	if !ok {
		return false, false, nil
	}
	keyBytes := []byte(key)
	if c.filter.Contains(keyBytes) {
		c.dup++
		return true, true, nil
	}
	if err := c.filter.Add(keyBytes); err != nil {
		return false, false, err
	}
	c.novel++
	return false, true, nil
}

// Remove drops a previously inserted key from the set when present.
func (c *CountingClassifier) Remove(line string) (removed bool, ok bool) {
	key, ok := c.keyFn(line)
	if !ok {
		return false, false
	}
	if c.filter.Remove([]byte(key)) {
		c.removed++
		return true, true
	}
	return false, true
}

// Filter returns the underlying counting Bloom filter.
func (c *CountingClassifier) Filter() *bloom.CountingFilter { return c.filter }

// Novel returns the number of first-seen lines.
func (c *CountingClassifier) Novel() int { return c.novel }

// Duplicates returns the number of duplicate lines seen.
func (c *CountingClassifier) Duplicates() int { return c.dup }

// Removed returns the number of successful Remove calls.
func (c *CountingClassifier) Removed() int { return c.removed }
