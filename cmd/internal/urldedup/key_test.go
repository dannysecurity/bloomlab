package urldedup

import "testing"

func TestKeyWithoutOptions(t *testing.T) {
	key, ok := Key("  https://Example.com/  ", Options{})
	if !ok || key != "https://Example.com/" {
		t.Fatalf("Key() = %q, ok=%v; want %q, true", key, ok, "https://Example.com/")
	}

	_, ok = Key("   ", Options{})
	if ok {
		t.Fatal("blank line should yield ok=false")
	}
}

func TestKeyNormalize(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "https default port and path trim", in: "https://Example.COM:443/path/", want: "https://example.com/path"},
		{name: "http default port", in: "http://HOST:80/a/", want: "http://host/a"},
		{name: "drop fragment", in: "HTTPS://a.test#frag", want: "https://a.test/"},
		{name: "schemeless host", in: "example.com/page", want: "https://example.com/page"},
		{name: "unparseable passthrough", in: "not a url at all", want: "not a url at all"},
		{name: "already canonical", in: "https://a.test/", want: "https://a.test/"},
		{name: "scheme relative", in: "//Example.com/path/", want: "https://example.com/path"},
		{name: "scheme relative default port", in: "//HOST:443/a", want: "https://host/a"},
	}

	opts := Options{Normalize: true}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := Key(tc.in, opts)
			if !ok {
				t.Fatalf("Key(%q) ok=false", tc.in)
			}
			if got != tc.want {
				t.Errorf("Key(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestKeyStripQuery(t *testing.T) {
	opts := Options{Normalize: true, StripQuery: true}

	key, ok := Key("https://a.test/page?x=1&y=2", opts)
	if !ok || key != "https://a.test/page" {
		t.Fatalf("Key() = %q, ok=%v; want https://a.test/page, true", key, ok)
	}

	dup, ok := Key("https://a.test/page?z=9", opts)
	if !ok || dup != key {
		t.Fatalf("same path different query should match: got %q want %q", dup, key)
	}
}

func TestKeyStripTracking(t *testing.T) {
	opts := Options{Normalize: true, StripTracking: true}

	first, ok := Key("https://a.test/page?utm_source=email&id=42", opts)
	if !ok {
		t.Fatal("expected ok=true")
	}
	if first != "https://a.test/page?id=42" {
		t.Fatalf("Key() = %q, want https://a.test/page?id=42", first)
	}

	second, ok := Key("https://a.test/page?utm_campaign=spring&id=42", opts)
	if !ok || second != first {
		t.Fatalf("tracking-only diff should match: got %q want %q", second, first)
	}

	third, ok := Key("https://a.test/page?id=99", opts)
	if !ok || third == first {
		t.Fatalf("different retained query should not match: got %q first=%q", third, first)
	}
}

func TestKeyDomainOnly(t *testing.T) {
	opts := Options{Normalize: true, DomainOnly: true}

	host, ok := Key("https://Example.com/path/a", opts)
	if !ok || host != "example.com" {
		t.Fatalf("Key() = %q, ok=%v; want example.com, true", host, ok)
	}

	dup, ok := Key("https://example.com/other", opts)
	if !ok || dup != host {
		t.Fatalf("same host different path should match: got %q want %q", dup, host)
	}

	other, ok := Key("https://other.test/", opts)
	if !ok || other == host {
		t.Fatalf("different host should not match: got %q host=%q", other, host)
	}
}

func TestKeyDomainOnlyUnparseable(t *testing.T) {
	_, ok := Key("http://", Options{DomainOnly: true})
	if ok {
		t.Fatal("URL without host with -domain-only should yield ok=false")
	}
}

func TestKeyCombinedStripTrackingAndQuery(t *testing.T) {
	opts := Options{Normalize: true, StripTracking: true, StripQuery: true}

	a, ok := Key("https://a.test/x?utm_source=1&keep=1", opts)
	if !ok || a != "https://a.test/x" {
		t.Fatalf("Key() = %q, want https://a.test/x", a)
	}

	b, ok := Key("https://a.test/x?keep=2", opts)
	if !ok || b != a {
		t.Fatalf("strip-query should ignore all query params: got %q want %q", b, a)
	}
}
