package streamflags

import (
	"flag"

	"github.com/dannysecurity/bloomlab/dedup"
)

// Flags holds shared CLI options for stream dedup tools.
type Flags struct {
	Quiet      *bool
	NovelOnly  *bool
	IgnoreCase *bool
	JSON       *bool
}

// Register binds stream output flags shared by streamdedup, countingdedup, and urldedup.
func Register() *Flags {
	return &Flags{
		Quiet:      flag.Bool("quiet", false, "print summary only"),
		NovelOnly:  flag.Bool("novel-only", false, "emit first-seen lines only"),
		IgnoreCase: flag.Bool("ignore-case", false, "compare lines case-insensitively"),
		JSON:       flag.Bool("json", false, "emit one JSON object per line on stdout"),
	}
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
