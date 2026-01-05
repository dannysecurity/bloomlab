package streamdedup

import (
	"io"

	"github.com/dannysecurity/bloomlab/dedup"
)

// Run reads lines from in, classifies each with d, and writes results.
func Run(d *Deduper, in io.Reader, opts RunOptions) error {
	return dedup.Run(d.Classifier, in, opts)
}
