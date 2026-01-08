package filterflags

import (
	"flag"
	"strings"
	"testing"

	"github.com/dannysecurity/bloomlab/bloom"
)

func TestFlagsConfigExplicitSizing(t *testing.T) {
	flag.CommandLine = flag.NewFlagSet("test", flag.ContinueOnError)
	f := Register(1000)
	if err := flag.CommandLine.Parse([]string{"-m", "512", "-k", "6", "-hash", "xxhash"}); err != nil {
		t.Fatal(err)
	}

	cfg, err := f.Config()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Mode() != bloom.SizingExplicit {
		t.Fatalf("Mode() = %v, want explicit", cfg.Mode())
	}
	if cfg.Bits != 512 || cfg.HashCount != 6 {
		t.Fatalf("explicit sizing = m=%d k=%d, want 512 and 6", cfg.Bits, cfg.HashCount)
	}
	if cfg.Hash.Strategy != bloom.HashXXHash {
		t.Fatalf("Hash.Strategy = %v, want xxhash", cfg.Hash.Strategy)
	}
}

func TestFlagsConfigWithBounds(t *testing.T) {
	flag.CommandLine = flag.NewFlagSet("test", flag.ContinueOnError)
	f := Register(1000)
	if err := flag.CommandLine.Parse([]string{"-min-bits", "256", "-max-k", "8"}); err != nil {
		t.Fatal(err)
	}

	cfg, err := f.Config()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.MinBits != 256 || cfg.MaxHashCount != 8 {
		t.Fatalf("bounds = minBits=%d maxK=%d, want 256 and 8", cfg.MinBits, cfg.MaxHashCount)
	}
}

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

func TestFlagsConfigTable(t *testing.T) {
	tests := []struct {
		name    string
		counting bool
		args    []string
		check   func(t *testing.T, cfg bloom.Config)
		wantErr string
	}{
		{
			name: "default fnv strategy",
			args: nil,
			check: func(t *testing.T, cfg bloom.Config) {
				if cfg.Hash.Strategy != bloom.HashFNV {
					t.Fatalf("Hash.Strategy = %v, want fnv", cfg.Hash.Strategy)
				}
			},
		},
		{
			name: "xxhash with seed and bounds",
			args: []string{"-hash", "xxhash", "-seed", "42", "-min-bits", "512", "-max-k", "12"},
			check: func(t *testing.T, cfg bloom.Config) {
				if cfg.Hash.Strategy != bloom.HashXXHash || cfg.Hash.Seed != 42 {
					t.Fatalf("Hash = %+v, want xxhash seed=42", cfg.Hash)
				}
				if cfg.MinBits != 512 || cfg.MaxHashCount != 12 {
					t.Fatalf("bounds = minBits=%d maxK=%d, want 512 and 12", cfg.MinBits, cfg.MaxHashCount)
				}
			},
		},
		{
			name: "wyhash strategy",
			args: []string{"-hash", "wyhash"},
			check: func(t *testing.T, cfg bloom.Config) {
				if cfg.Hash.Strategy != bloom.HashWyhash {
					t.Fatalf("Hash.Strategy = %v, want wyhash", cfg.Hash.Strategy)
				}
			},
		},
		{
			name:    "invalid hash strategy",
			args:    []string{"-hash", "sha256"},
			wantErr: "unknown hash strategy",
		},
		{
			name:     "counting 16-bit counter",
			counting: true,
			args:     []string{"-counter-width", "16"},
			check: func(t *testing.T, cfg bloom.Config) {
				if cfg.CounterWidth != 16 {
					t.Fatalf("CounterWidth = %d, want 16", cfg.CounterWidth)
				}
			},
		},
		{
			name:     "counting invalid counter width",
			counting: true,
			args:     []string{"-counter-width", "32"},
			wantErr:  "counter-width must be 8 or 16",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag.CommandLine = flag.NewFlagSet("test", flag.ContinueOnError)
			var f *Flags
			if tt.counting {
				f = RegisterCounting(1000)
			} else {
				f = Register(1000)
			}
			if err := flag.CommandLine.Parse(tt.args); err != nil {
				t.Fatal(err)
			}

			cfg, err := f.Config()
			if tt.wantErr != "" {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("Config() error = %q, want substring %q", err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if tt.check != nil {
				tt.check(t, cfg)
			}
		})
	}
}
