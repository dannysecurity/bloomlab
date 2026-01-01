package main

import (
	"testing"

	"github.com/dannysecurity/bloomlab/cmd/internal/urldedup"
)

func TestKeyWithoutNormalize(t *testing.T) {
	key, ok := urldedup.Key("  https://Example.com/  ", urldedup.Options{})
	if !ok || key != "https://Example.com/" {
		t.Fatalf("Key() = %q, ok=%v; want %q, true", key, ok, "https://Example.com/")
	}

	_, ok = urldedup.Key("   ", urldedup.Options{})
	if ok {
		t.Fatal("blank line should yield ok=false")
	}
}

func TestKeyNormalize(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"https://Example.COM:443/path/", "https://example.com/path"},
		{"http://HOST:80/a/", "http://host/a"},
		{"HTTPS://a.test#frag", "https://a.test/"},
		{"example.com/page", "https://example.com/page"},
		{"not a url at all", "not a url at all"},
		{"https://a.test/", "https://a.test/"},
		{"//Example.com/path/", "https://example.com/path"},
		{"//HOST:443/a", "https://host/a"},
	}

	opts := urldedup.Options{Normalize: true}
	for _, tc := range tests {
		got, ok := urldedup.Key(tc.in, opts)
		if !ok {
			t.Fatalf("Key(%q) ok=false", tc.in)
		}
		if got != tc.want {
			t.Errorf("Key(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestClassifyWithNormalize(t *testing.T) {
	f, err := bloomNewTestFilter()
	if err != nil {
		t.Fatal(err)
	}

	opts := dedupOptions{Normalize: true}

	dup, ok := classify(f, "HTTPS://Example.com:443/path/", opts)
	if !ok || dup {
		t.Fatalf("first canonical visit: ok=%v dup=%v", ok, dup)
	}

	dup, ok = classify(f, "https://example.com/path", opts)
	if !ok || !dup {
		t.Fatalf("equivalent url should be duplicate: ok=%v dup=%v", ok, dup)
	}

	dup, ok = classify(f, "https://other.test", opts)
	if !ok || dup {
		t.Fatalf("different host: ok=%v dup=%v", ok, dup)
	}

	dup, ok = classify(f, "//example.com/path", opts)
	if !ok || !dup {
		t.Fatalf("protocol-relative url should match https equivalent: ok=%v dup=%v", ok, dup)
	}
}

func TestClassifyStripTracking(t *testing.T) {
	f, err := bloomNewTestFilter()
	if err != nil {
		t.Fatal(err)
	}

	opts := dedupOptions{Normalize: true, StripTracking: true}

	dup, ok := classify(f, "https://a.test/page?utm_source=email&id=1", opts)
	if !ok || dup {
		t.Fatalf("first visit: ok=%v dup=%v", ok, dup)
	}

	dup, ok = classify(f, "https://a.test/page?fbclid=xyz&id=1", opts)
	if !ok || !dup {
		t.Fatalf("tracking-only diff should duplicate: ok=%v dup=%v", ok, dup)
	}
}

func TestClassifyDomainOnly(t *testing.T) {
	f, err := bloomNewTestFilter()
	if err != nil {
		t.Fatal(err)
	}

	opts := dedupOptions{Normalize: true, DomainOnly: true}

	dup, ok := classify(f, "https://a.test/one", opts)
	if !ok || dup {
		t.Fatalf("first visit: ok=%v dup=%v", ok, dup)
	}

	dup, ok = classify(f, "https://a.test/two", opts)
	if !ok || !dup {
		t.Fatalf("same host should duplicate: ok=%v dup=%v", ok, dup)
	}
}
