package dedup

import "github.com/dannysecurity/bloomlab/bloom"

// Classifier reports novel vs duplicate lines using a standard Bloom filter.
type Classifier struct {
	filter *bloom.Filter
	keyFn  KeyFunc
	novel  int
	dup    int
}

// NewClassifier builds a Classifier. keyFn defaults to TrimKey when nil.
func NewClassifier(f *bloom.Filter, keyFn KeyFunc) *Classifier {
	if keyFn == nil {
		keyFn = TrimKey
	}
	return &Classifier{filter: f, keyFn: keyFn}
}

// Classify reports whether line is a duplicate and inserts it when novel.
func (c *Classifier) Classify(line string) (duplicate bool, ok bool) {
	key, ok := c.keyFn(line)
	if !ok {
		return false, false
	}
	keyBytes := []byte(key)
	if c.filter.Contains(keyBytes) {
		c.dup++
		return true, true
	}
	c.filter.Add(keyBytes)
	c.novel++
	return false, true
}

// Filter returns the underlying Bloom filter.
func (c *Classifier) Filter() *bloom.Filter { return c.filter }

// Novel returns the number of first-seen lines.
func (c *Classifier) Novel() int { return c.novel }

// Duplicates returns the number of duplicate lines seen.
func (c *Classifier) Duplicates() int { return c.dup }
