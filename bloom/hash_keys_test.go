package bloom

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseKeyDistribution(t *testing.T) {
	tests := []struct {
		in   string
		want KeyDistribution
	}{
		{"", KeySequential},
		{"sequential", KeySequential},
		{"seq", KeySequential},
		{"url", KeyURL},
		{"uuid", KeyUUID},
		{"fixed32", KeyFixed32},
		{"samples", KeyFromSamples},
	}
	for _, tt := range tests {
		got, err := ParseKeyDistribution(tt.in)
		if err != nil {
			t.Fatalf("ParseKeyDistribution(%q) error: %v", tt.in, err)
		}
		if got != tt.want {
			t.Errorf("ParseKeyDistribution(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
	if _, err := ParseKeyDistribution("json"); err == nil {
		t.Fatal("expected error for unknown distribution")
	}
}

func TestKeyForDistributionShapes(t *testing.T) {
	seq := KeyForDistribution(KeySequential, "probe")(42)
	if string(seq) != "probe-42" {
		t.Fatalf("sequential key = %q, want probe-42", seq)
	}

	url := KeyForDistribution(KeyURL, "ref")(3)
	if !strings.HasPrefix(string(url), "https://") {
		t.Fatalf("url key = %q, want https prefix", url)
	}
	if !strings.Contains(string(url), "ref") {
		t.Fatalf("url key = %q, want ref prefix in path/query", url)
	}

	uuid := KeyForDistribution(KeyUUID, "id")(1)
	if len(uuid) != 36 || uuid[8] != '-' || uuid[13] != '-' {
		t.Fatalf("uuid key = %q, want 36-char dashed format", uuid)
	}

	fixed := KeyForDistribution(KeyFixed32, "bin")(99)
	if len(fixed) != 32 {
		t.Fatalf("fixed32 key len = %d, want 32", len(fixed))
	}
	if fixed[27] != 99 {
		t.Fatalf("fixed32 index byte = %d, want 99", fixed[27])
	}
}

func TestKeyForSamplesCycles(t *testing.T) {
	keys := [][]byte{[]byte("alpha"), []byte("beta")}
	keyFor := KeyForSamples(keys)
	if string(keyFor(0)) != "alpha" || string(keyFor(1)) != "beta" || string(keyFor(2)) != "alpha" {
		t.Fatalf("unexpected cycle: %q %q %q", keyFor(0), keyFor(1), keyFor(2))
	}
}

func TestLoadKeyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "keys.txt")
	content := "https://example.com/a\n\nhttps://example.com/b\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	keys, err := LoadKeyFile(path, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 2 {
		t.Fatalf("got %d keys, want 2", len(keys))
	}
	if string(keys[0]) != "https://example.com/a" {
		t.Fatalf("keys[0] = %q", keys[0])
	}

	if _, err := LoadKeyFile(path, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadKeyFile(filepath.Join(dir, "missing.txt"), 10); err == nil {
		t.Fatal("expected error for missing file")
	}

	emptyPath := filepath.Join(dir, "empty.txt")
	if err := os.WriteFile(emptyPath, []byte("\n\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadKeyFile(emptyPath, 10); err == nil {
		t.Fatal("expected error for empty key file")
	}
}

func TestRecommendHasherURLDistribution(t *testing.T) {
	const m = 4096
	const k = 7
	const samples = 15_000

	opts := TuneOptions{
		M:            m,
		K:            k,
		Samples:      samples,
		Distribution: KeyURL,
		KeyFor:       KeyForDistribution(KeyURL, "urltune"),
	}
	report := RecommendHasher(opts, AllStrategies(), DefaultTuneSeeds())

	var fnvChi float64
	for _, score := range report.Strategies {
		if score.Strategy == HashFNV {
			fnvChi = score.Spread.ChiSquared
			break
		}
	}
	if fnvChi == 0 {
		t.Fatal("missing FNV score in report")
	}
	if report.Best.Strategy == HashFNV {
		t.Fatalf("expected non-FNV best on URL keys, got fnv chi²=%.1f", fnvChi)
	}
	if report.Best.Spread.ChiSquared >= fnvChi {
		t.Fatalf("best chi² %.1f should beat FNV %.1f on URL keys",
			report.Best.Spread.ChiSquared, fnvChi)
	}
}

func TestTuneOptionsFromConfigWithDist(t *testing.T) {
	cfg := TargetConfig(5000, 0.01)

	opts, err := TuneOptionsFromConfigWithDist(cfg, 500, "probe", KeyURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	if opts.Distribution != KeyURL {
		t.Fatalf("distribution = %v, want url", opts.Distribution)
	}
	key := opts.KeyFor(0)
	if !strings.HasPrefix(string(key), "https://") {
		t.Fatalf("KeyFor(0) = %q, want URL shape", key)
	}

	_, err = TuneOptionsFromConfigWithDist(cfg, 500, "probe", KeyFromSamples, nil)
	if err == nil {
		t.Fatal("expected error for samples distribution without keys")
	}

	sampleKeys := [][]byte{[]byte("line-a"), []byte("line-b")}
	opts, err = TuneOptionsFromConfigWithDist(cfg, 500, "probe", KeyFromSamples, sampleKeys)
	if err != nil {
		t.Fatal(err)
	}
	if string(opts.KeyFor(1)) != "line-b" {
		t.Fatalf("KeyFor(1) = %q, want line-b", opts.KeyFor(1))
	}
}

func TestFormatTuningReportShowsDistribution(t *testing.T) {
	opts := TuneOptions{
		M:            1024,
		K:            4,
		Samples:      1000,
		Distribution: KeyURL,
		KeyFor:       KeyForDistribution(KeyURL, "fmt"),
	}
	report := RecommendHasher(opts, []Strategy{HashMurmur3}, []uint64{0})
	text := FormatTuningReport(report)
	if !strings.Contains(text, "key-dist=url") {
		t.Fatalf("report missing distribution: %q", text)
	}
}
