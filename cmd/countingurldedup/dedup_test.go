package main

import (
	"testing"

	"github.com/dannysecurity/bloomlab/bloom"
)

func testCountingFilter(t *testing.T) *bloom.CountingFilter {
	t.Helper()
	cf, err := bloom.NewCountingFilter(bloom.TargetConfig(100, 0.01))
	if err != nil {
		t.Fatal(err)
	}
	return cf
}

func TestClassify(t *testing.T) {
	cf := testCountingFilter(t)
	opts := dedupOptions{Normalize: true}

	dup, ok, err := classify(cf, "HTTPS://Example.com:443/path/", opts)
	if err != nil || !ok || dup {
		t.Fatalf("first visit: err=%v ok=%v dup=%v, want ok=true dup=false", err, ok, dup)
	}

	dup, ok, err = classify(cf, "https://example.com/path", opts)
	if err != nil || !ok || !dup {
		t.Fatalf("canonical duplicate: err=%v ok=%v dup=%v, want ok=true dup=true", err, ok, dup)
	}

	dup, ok, err = classify(cf, "https://other.test", opts)
	if err != nil || !ok || dup {
		t.Fatalf("novel url: err=%v ok=%v dup=%v, want ok=true dup=false", err, ok, dup)
	}

	dup, ok, err = classify(cf, "   ", opts)
	if err != nil || ok || dup {
		t.Fatalf("blank line: err=%v ok=%v dup=%v, want ok=false dup=false", err, ok, dup)
	}
}

func TestClassifyAndRemove(t *testing.T) {
	cf := testCountingFilter(t)
	opts := dedupOptions{Normalize: true, StripTracking: true}

	dup, ok, err := classify(cf, "https://a.test/page?utm_source=x", opts)
	if err != nil || !ok || dup {
		t.Fatalf("first visit: err=%v ok=%v dup=%v, want ok=true dup=false", err, ok, dup)
	}

	dup, ok, err = classify(cf, "https://a.test/page?fbclid=y", opts)
	if err != nil || !ok || !dup {
		t.Fatalf("tracking duplicate: err=%v ok=%v dup=%v, want ok=true dup=true", err, ok, dup)
	}

	removed, ok := remove(cf, "https://a.test/page", opts)
	if !ok || !removed {
		t.Fatalf("remove: ok=%v removed=%v, want ok=true removed=true", ok, removed)
	}

	dup, ok, err = classify(cf, "https://a.test/page?utm_source=z", opts)
	if err != nil || !ok || dup {
		t.Fatalf("after remove: err=%v ok=%v dup=%v, want ok=true dup=false", err, ok, dup)
	}
}
