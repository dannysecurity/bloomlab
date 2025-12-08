package main

import "testing"

func TestCanonicalKeyWithoutNormalize(t *testing.T) {
	key, ok := canonicalKey("  https://Example.com/  ", false)
	if !ok || key != "https://Example.com/" {
		t.Fatalf("canonicalKey() = %q, ok=%v; want %q, true", key, ok, "https://Example.com/")
	}

	_, ok = canonicalKey("   ", false)
	if ok {
		t.Fatal("blank line should yield ok=false")
	}
}

func TestNormalizeURL(t *testing.T) {
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
	}

	for _, tc := range tests {
		got := normalizeURL(tc.in)
		if got != tc.want {
			t.Errorf("normalizeURL(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestClassifyWithNormalize(t *testing.T) {
	f, err := bloomNewTestFilter()
	if err != nil {
		t.Fatal(err)
	}

	opts := dedupOptions{normalize: true}

	dup, ok := classify(f, "HTTP://Example.com:80/path/", opts)
	if !ok || dup {
		t.Fatalf("first canonical visit: ok=%v dup=%v", ok, dup)
	}

	dup, ok = classify(f, "http://example.com/path", opts)
	if !ok || !dup {
		t.Fatalf("equivalent url should be duplicate: ok=%v dup=%v", ok, dup)
	}

	dup, ok = classify(f, "https://other.test", opts)
	if !ok || dup {
		t.Fatalf("different host: ok=%v dup=%v", ok, dup)
	}
}
