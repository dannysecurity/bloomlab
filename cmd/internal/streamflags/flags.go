package streamflags

import (
	"errors"
	"flag"

	"github.com/dannysecurity/bloomlab/dedup"
)

// Flags holds shared CLI options for stream dedup tools.
type Flags struct {
	Quiet      *bool
	NovelOnly  *bool
	DupOnly    *bool
	IgnoreCase *bool
	JSON       *bool
}

// Register binds stream output flags shared by stream dedup CLIs.
func Register() *Flags {
	return &Flags{
		Quiet:      flag.Bool("quiet", false, "print summary only"),
		NovelOnly:  flag.Bool("novel-only", false, "emit first-seen lines only"),
		DupOnly:    flag.Bool("dup-only", false, "emit duplicate lines only"),
		IgnoreCase: flag.Bool("ignore-case", false, "compare lines case-insensitively"),
		JSON:       flag.Bool("json", false, "emit one JSON object per line on stdout"),
	}
}

// Validate checks for incompatible flag combinations after parsing.
func (f *Flags) Validate() error {
	if *f.NovelOnly && *f.DupOnly {
		return errors.New("-novel-only and -dup-only are mutually exclusive")
	}
	return nil
}

// RunOptions builds dedup.RunOptions from parsed flag values.
func (f *Flags) RunOptions() dedup.RunOptions {
	format := dedup.FormatText
	if *f.JSON {
		format = dedup.FormatJSON
	}
	return dedup.RunOptions{
		Quiet:     *f.Quiet,
		NovelOnly: *f.NovelOnly,
		DupOnly:   *f.DupOnly,
		Format:    format,
	}
}

// KeyFunc returns the line key function, optionally lowercasing keys.
func (f *Flags) KeyFunc() dedup.KeyFunc {
	if *f.IgnoreCase {
		return dedup.IgnoreCaseKey
	}
	return dedup.TrimKey
}
