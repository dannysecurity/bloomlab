package streamdedup

import (
	"github.com/dannysecurity/bloomlab/bloom"
	"github.com/dannysecurity/bloomlab/dedup"
)

// KeyFunc maps a line to a dedup key. Return ok=false to skip the line.
type KeyFunc = dedup.KeyFunc

// TrimKey trims whitespace and skips blank lines.
var TrimKey = dedup.TrimKey

// IgnoreCaseKey trims whitespace, lowercases, and skips blank lines.
var IgnoreCaseKey = dedup.IgnoreCaseKey

// Format selects per-line output encoding.
type Format = dedup.Format

const (
	FormatText = dedup.FormatText
	FormatJSON = dedup.FormatJSON
)

// RunOptions configures stream processing and output.
type RunOptions = dedup.RunOptions

// Deduper classifies stdin lines as novel or duplicate using a Bloom filter.
type Deduper struct {
	*dedup.Classifier
}

// New builds a Deduper. keyFn defaults to TrimKey when nil.
func New(f *bloom.Filter, keyFn KeyFunc) *Deduper {
	return &Deduper{Classifier: dedup.NewClassifier(f, keyFn)}
}

// Filter returns the underlying Bloom filter.
func (d *Deduper) Filter() *bloom.Filter { return d.Classifier.Filter() }
