package benchcompare

import (
	"strings"
	"testing"

	"github.com/dannysecurity/bloomlab/bloom"
)

func TestParseFPRRatesTable(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    []float64
		wantErr string
	}{
		{
			name: "default sweep spacing",
			raw:  "0.001,0.01,0.1",
			want: []float64{0.001, 0.01, 0.1},
		},
		{
			name: "whitespace tolerant",
			raw:  " 0.001 , 0.01 , 0.1 ",
			want: []float64{0.001, 0.01, 0.1},
		},
		{
			name: "skip empty segments",
			raw:  "0.01,,0.05",
			want: []float64{0.01, 0.05},
		},
		{
			name: "single rate",
			raw:  "0.05",
			want: []float64{0.05},
		},
		{
			name:    "empty string",
			raw:     "",
			wantErr: "no FPR rates",
		},
		{
			name:    "only commas and spaces",
			raw:     "  ,  , ",
			wantErr: "no FPR rates",
		},
		{
			name: "zero rate parses",
			raw:  "0,0.01",
			want: []float64{0, 0.01},
		},
		{
			name: "negative rate parses",
			raw:  "-0.01",
			want: []float64{-0.01},
		},
		{
			name: "rate equals one parses",
			raw:  "1.0",
			want: []float64{1.0},
		},
		{
			name:    "not a float",
			raw:     "not-a-float",
			wantErr: "invalid FPR rate",
		},
		{
			name:    "mixed valid and invalid",
			raw:     "0.01,xy",
			wantErr: "invalid FPR rate",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseFPRRates(tt.raw)
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("ParseFPRRates(%q) succeeded, want error containing %q", tt.raw, tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("ParseFPRRates(%q) error = %v, want substring %q", tt.raw, err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseFPRRates(%q) unexpected error: %v", tt.raw, err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("got %d rates %v, want %d %v", len(got), got, len(tt.want), tt.want)
			}
			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Errorf("rates[%d] = %v, want %v", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestCompareFPRSweepValidationTable(t *testing.T) {
	cfg := Config{Bloom: bloom.TargetFilter(500, 0.01), LookupRepeats: 1}
	tests := []struct {
		name    string
		rates   []float64
		wantErr string
	}{
		{
			name:  "valid sweep",
			rates: []float64{0.001, 0.01, 0.1},
		},
		{
			name:    "nil rates",
			rates:   nil,
			wantErr: "requires at least one rate",
		},
		{
			name:    "empty slice",
			rates:   []float64{},
			wantErr: "requires at least one rate",
		},
		{
			name:    "zero rate",
			rates:   []float64{0, 0.01},
			wantErr: "must be in (0, 1)",
		},
		{
			name:    "negative rate",
			rates:   []float64{-0.01},
			wantErr: "must be in (0, 1)",
		},
		{
			name:    "rate equals one",
			rates:   []float64{1.0},
			wantErr: "must be in (0, 1)",
		},
		{
			name:    "rate above one",
			rates:   []float64{1.5},
			wantErr: "must be in (0, 1)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := CompareFPRSweep(cfg, tt.rates)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("CompareFPRSweep() unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("CompareFPRSweep() succeeded, want error containing %q", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("CompareFPRSweep() error = %v, want substring %q", err, tt.wantErr)
			}
		})
	}
}

func TestDefaultFPRSweepRatesTable(t *testing.T) {
	want := []float64{0.001, 0.01, 0.1}
	if len(DefaultFPRSweepRates) != len(want) {
		t.Fatalf("DefaultFPRSweepRates len = %d, want %d", len(DefaultFPRSweepRates), len(want))
	}
	for i := range want {
		if DefaultFPRSweepRates[i] != want[i] {
			t.Errorf("DefaultFPRSweepRates[%d] = %v, want %v", i, DefaultFPRSweepRates[i], want[i])
		}
	}

	rates, err := ParseFPRRates("0.001,0.01,0.1")
	if err != nil {
		t.Fatal(err)
	}
	for i := range want {
		if rates[i] != DefaultFPRSweepRates[i] {
			t.Errorf("parsed[%d] = %v, want default %v", i, rates[i], DefaultFPRSweepRates[i])
		}
	}
}
