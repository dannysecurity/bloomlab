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
	if opts.Quiet || opts.NovelOnly || opts.Format != dedup.FormatText {
		t.Fatalf("RunOptions() = %+v, want quiet=false novelOnly=false text format", opts)
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
	if !opts.Quiet || !opts.NovelOnly || opts.Format != dedup.FormatJSON {
		t.Fatalf("RunOptions() = %+v, want quiet novel-only json", opts)
	}
	key, ok := f.KeyFunc()("  HeLLo  ")
	if !ok || key != "hello" {
		t.Fatalf("KeyFunc() = %q ok=%v, want hello", key, ok)
	}
}
