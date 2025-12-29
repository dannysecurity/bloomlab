package filterflags

import (
	"flag"
	"testing"

	"github.com/dannysecurity/bloomlab/bloom"
)

func TestFlagsConfig(t *testing.T) {
	flag.CommandLine = flag.NewFlagSet("test", flag.ContinueOnError)
	f := Register(1000)
	if err := flag.CommandLine.Parse([]string{"-hash", "murmur3", "-seed", "9"}); err != nil {
		t.Fatal(err)
	}

	cfg, err := f.Config()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Hash.Strategy != bloom.HashMurmur3 || cfg.Hash.Seed != 9 {
		t.Fatalf("Hash = %+v, want murmur3 seed=9", cfg.Hash)
	}
	if cfg.ExpectedCapacity != 1000 || cfg.FalsePositiveRate != 0.01 {
		t.Fatalf("sizing = n=%d p=%g", cfg.ExpectedCapacity, cfg.FalsePositiveRate)
	}
}

func TestFlagsCountingConfigWideCounterWidth(t *testing.T) {
	flag.CommandLine = flag.NewFlagSet("test", flag.ContinueOnError)
	f := RegisterCounting(1000)
	if err := flag.CommandLine.Parse([]string{"-counter-width", "16"}); err != nil {
		t.Fatal(err)
	}

	cfg, err := f.Config()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.CounterWidth != 16 {
		t.Fatalf("CounterWidth = %d, want 16", cfg.CounterWidth)
	}
}

func TestFlagsCountingConfigInvalidCounterWidth(t *testing.T) {
	flag.CommandLine = flag.NewFlagSet("test", flag.ContinueOnError)
	f := RegisterCounting(1000)
	if err := flag.CommandLine.Parse([]string{"-counter-width", "32"}); err != nil {
		t.Fatal(err)
	}

	if _, err := f.Config(); err == nil {
		t.Fatal("expected error for invalid counter width")
	}
}
