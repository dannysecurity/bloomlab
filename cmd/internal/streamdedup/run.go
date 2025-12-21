package streamdedup

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// Format selects per-line output encoding.
type Format int

const (
	FormatText Format = iota
	FormatJSON
)

// RunOptions configures stream processing and output.
type RunOptions struct {
	Quiet  bool
	Format Format
	Out    io.Writer
	ErrOut io.Writer
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

// Run reads lines from in, classifies each with d, and writes results.
func Run(d *Deduper, in io.Reader, opts RunOptions) error {
	scanner := bufio.NewScanner(in)
	out := opts.out()
	for scanner.Scan() {
		line := scanner.Text()
		isDup, ok := d.Classify(line)
		if !ok || opts.Quiet {
			continue
		}
		if err := writeResult(out, opts.Format, isDup, line); err != nil {
			return err
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read input: %w", err)
	}
	writeSummary(opts.errOut(), d)
	return nil
}

func writeResult(w io.Writer, format Format, duplicate bool, line string) error {
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

func writeSummary(w io.Writer, d *Deduper) {
	f := d.Filter()
	fmt.Fprintf(w, "novel: %d, duplicates: %d, inserts: %d, fill: %.2f%%, theory FPR: %.4f%%\n",
		d.Novel(), d.Duplicates(), f.ApproximateCount(), f.FillRatio()*100, f.TheoryFPR()*100)
}
