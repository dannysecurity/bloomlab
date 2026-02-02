package streamflags

import (
	"flag"
	"testing"

	"github.com/dannysecurity/bloomlab/dedup"
)

func TestRunOptionsDefaults(t *testing.T) {
	flag.CommandLine = flag.NewFlagSet("test", flag.ContinueOnError)
	f := Register()
	if err := flag.CommandLine.Parse(nil); err != nil {
		t.Fatal(err)
	}

	opts := f.RunOptions()
	if opts.Quiet || opts.NovelOnly || opts.DupOnly || opts.Format != dedup.FormatText {
		t.Fatalf("RunOptions() = %+v, want quiet=false novelOnly=false dupOnly=false text format", opts)
	}
	key, ok := f.KeyFunc()("  Hello  ")
	if !ok || key != "Hello" {
		t.Fatalf("KeyFunc() = %q ok=%v, want Hello", key, ok)
	}
}

func TestRunOptionsJSONAndIgnoreCase(t *testing.T) {
	flag.CommandLine = flag.NewFlagSet("test", flag.ContinueOnError)
	f := Register()
	if err := flag.CommandLine.Parse([]string{"-json", "-ignore-case", "-quiet", "-novel-only"}); err != nil {
		t.Fatal(err)
	}

	opts := f.RunOptions()
	if !opts.Quiet || !opts.NovelOnly || opts.DupOnly || opts.Format != dedup.FormatJSON {
		t.Fatalf("RunOptions() = %+v, want quiet novel-only json", opts)
	}
	key, ok := f.KeyFunc()("  HeLLo  ")
	if !ok || key != "hello" {
		t.Fatalf("KeyFunc() = %q ok=%v, want hello", key, ok)
	}
}

func TestRunOptionsDupOnly(t *testing.T) {
	flag.CommandLine = flag.NewFlagSet("test", flag.ContinueOnError)
	f := Register()
	if err := flag.CommandLine.Parse([]string{"-dup-only"}); err != nil {
		t.Fatal(err)
	}

	opts := f.RunOptions()
	if !opts.DupOnly || opts.NovelOnly {
		t.Fatalf("RunOptions() = %+v, want dupOnly=true novelOnly=false", opts)
	}
}

func TestValidateMutuallyExclusiveOutputModes(t *testing.T) {
	flag.CommandLine = flag.NewFlagSet("test", flag.ContinueOnError)
	f := Register()
	if err := flag.CommandLine.Parse([]string{"-novel-only", "-dup-only"}); err != nil {
		t.Fatal(err)
	}
	if err := f.Validate(); err == nil {
		t.Fatal("Validate() = nil, want error for -novel-only and -dup-only together")
	}
}
