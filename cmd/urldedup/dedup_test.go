package main

import (
	"testing"

	"github.com/dannysecurity/bloomlab/bloom"
)

func bloomNewTestFilter() (*bloom.Filter, error) {
	return bloom.NewFilter(bloom.TargetConfig(100, 0.01))
}

func TestClassify(t *testing.T) {
	f, err := bloomNewTestFilter()
	if err != nil {
		t.Fatal(err)
	}

	opts := dedupOptions{}

	dup, ok := classify(f, "  https://example.com  ", opts)
	if ok != true || dup != false {
		t.Fatalf("first visit: ok=%v dup=%v, want ok=true dup=false", ok, dup)
	}

	dup, ok = classify(f, "https://example.com", opts)
	if ok != true || dup != true {
		t.Fatalf("repeat visit: ok=%v dup=%v, want ok=true dup=true", ok, dup)
	}

	dup, ok = classify(f, "https://other.test", opts)
	if ok != true || dup != false {
		t.Fatalf("novel url: ok=%v dup=%v, want ok=true dup=false", ok, dup)
	}

	dup, ok = classify(f, "   ", opts)
	if ok != false || dup != false {
		t.Fatalf("blank line: ok=%v dup=%v, want ok=false dup=false", ok, dup)
	}
}
