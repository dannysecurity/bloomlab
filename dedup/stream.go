package dedup

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// Format selects per-line output encoding.
type Format int

const (
	FormatText Format = iota
	FormatJSON
)

// RunOptions configures stream processing and output.
type RunOptions struct {
	Quiet     bool
	NovelOnly bool // emit first-seen lines only; skip duplicates
	DupOnly   bool // emit duplicate lines only; skip first-seen
	Format    Format
	Out       io.Writer
	ErrOut    io.Writer
}

func (o *RunOptions) out() io.Writer {
	if o.Out != nil {
		return o.Out
	}
	return os.Stdout
}

func (o *RunOptions) errOut() io.Writer {
	if o.ErrOut != nil {
		return o.ErrOut
	}
	return os.Stderr
}

type lineResult struct {
	Status string `json:"status"`
	Line   string `json:"line"`
}

// Run reads lines from in, classifies each with c, and writes results.
func Run(c *Classifier, in io.Reader, opts RunOptions) error {
	scanner := bufio.NewScanner(in)
	out := opts.out()
	for scanner.Scan() {
		line := scanner.Text()
		isDup, ok := c.Classify(line)
		if !ok || opts.Quiet || skipClassifyOutput(opts, isDup) {
			continue
		}
		if err := writeClassifyResult(out, opts.Format, isDup, line); err != nil {
			return err
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read input: %w", err)
	}
	writeClassifierSummary(opts.errOut(), c)
	return nil
}

// CountingRunOptions configures counting-filter stream processing.
type CountingRunOptions struct {
	RunOptions
	RemovePrefix string // lines with this prefix trigger Remove on the remainder
}

// RunCounting reads lines from in, classifying or removing keys via c.
// When RemovePrefix is non-empty, a line like "-item" removes "item" instead
// of classifying it.
func RunCounting(c *CountingClassifier, in io.Reader, opts CountingRunOptions) error {
	scanner := bufio.NewScanner(in)
	out := opts.out()
	prefix := opts.RemovePrefix
	for scanner.Scan() {
		line := scanner.Text()
		if prefix != "" && strings.HasPrefix(line, prefix) && len(line) > len(prefix) {
			removed, ok := c.Remove(line[len(prefix):])
			if !ok || opts.Quiet {
				continue
			}
			if err := writeRemoveResult(out, opts.Format, removed, line); err != nil {
				return err
			}
			continue
		}

		isDup, ok, err := c.Classify(line)
		if err != nil {
			return fmt.Errorf("classify %q: %w", line, err)
		}
		if !ok || opts.Quiet || skipClassifyOutput(opts.RunOptions, isDup) {
			continue
		}
		if err := writeClassifyResult(out, opts.Format, isDup, line); err != nil {
			return err
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read input: %w", err)
	}
	writeCountingSummary(opts.errOut(), c)
	return nil
}

func skipClassifyOutput(opts RunOptions, duplicate bool) bool {
	if opts.NovelOnly && duplicate {
		return true
	}
	if opts.DupOnly && !duplicate {
		return true
	}
	return false
}

func writeClassifyResult(w io.Writer, format Format, duplicate bool, line string) error {
	switch format {
	case FormatJSON:
		status := "new"
		if duplicate {
			status = "dup"
		}
		b, err := json.Marshal(lineResult{Status: status, Line: line})
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(w, "%s\n", b)
		return err
	default:
		if duplicate {
			_, err := fmt.Fprintf(w, "dup\t%s\n", line)
			return err
		}
		_, err := fmt.Fprintf(w, "new\t%s\n", line)
		return err
	}
}

func writeRemoveResult(w io.Writer, format Format, removed bool, line string) error {
	switch format {
	case FormatJSON:
		status := "removed"
		if !removed {
			status = "not-present"
		}
		b, err := json.Marshal(lineResult{Status: status, Line: line})
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(w, "%s\n", b)
		return err
	default:
		if removed {
			_, err := fmt.Fprintf(w, "removed\t%s\n", line)
			return err
		}
		_, err := fmt.Fprintf(w, "not-present\t%s\n", line)
		return err
	}
}

func writeClassifierSummary(w io.Writer, c *Classifier) {
	f := c.Filter()
	fmt.Fprintf(w, "novel: %d, duplicates: %d, inserts: %d, fill: %.2f%%, theory FPR: %.4f%%\n",
		c.Novel(), c.Duplicates(), f.ApproximateCount(), f.FillRatio()*100, f.TheoryFPR()*100)
}

func writeCountingSummary(w io.Writer, c *CountingClassifier) {
	f := c.Filter()
	fmt.Fprintf(w, "novel: %d, duplicates: %d, removed: %d, inserts: %d, fill: %.2f%%, theory FPR: %.4f%%\n",
		c.Novel(), c.Duplicates(), c.Removed(), f.ApproximateCount(), f.FillRatio()*100, f.TheoryFPR()*100)
}
