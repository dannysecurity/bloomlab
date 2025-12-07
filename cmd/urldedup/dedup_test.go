package main

import (
	"testing"

	"github.com/dannysecurity/bloomlab/bloom"
)

func TestClassify(t *testing.T) {
	f, err := bloom.NewFilter(bloom.TargetConfig(100, 0.01))
	if err != nil {
		t.Fatal(err)
	}

	dup, ok := classify(f, "  https://example.com  ")
	if ok != true || dup != false {
		t.Fatalf("first visit: ok=%v dup=%v, want ok=true dup=false", ok, dup)
	}

	dup, ok = classify(f, "https://example.com")
	if ok != true || dup != true {
		t.Fatalf("repeat visit: ok=%v dup=%v, want ok=true dup=true", ok, dup)
	}

	dup, ok = classify(f, "https://other.test")
	if ok != true || dup != false {
		t.Fatalf("novel url: ok=%v dup=%v, want ok=true dup=false", ok, dup)
	}

	dup, ok = classify(f, "   ")
	if ok != false || dup != false {
		t.Fatalf("blank line: ok=%v dup=%v, want ok=false dup=false", ok, dup)
	}
}
