package urldedup

import (
	"net/url"
	"testing"
	"testing/quick"
)

var normalizeQuick = &quick.Config{MaxCount: 200}

func TestParseURLTable(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantOK  bool
		wantStr string // expected u.String() when wantOK; empty when !wantOK
	}{
		{
			name:    "https with host and path",
			raw:     "https://Example.com/path",
			wantOK:  true,
			wantStr: "https://Example.com/path",
		},
		{
			name:    "schemeless host",
			raw:     "example.com/page",
			wantOK:  true,
			wantStr: "https://example.com/page",
		},
		{
			name:    "scheme relative",
			raw:     "//HOST/path/",
			wantOK:  true,
			wantStr: "//HOST/path/",
		},
		{
			name:    "http with explicit port",
			raw:     "http://host:8080/x",
			wantOK:  true,
			wantStr: "http://host:8080/x",
		},
		{
			name:    "ipv6 with port",
			raw:     "https://[2001:db8::1]:8443/x",
			wantOK:  true,
			wantStr: "https://[2001:db8::1]:8443/x",
		},
		{
			name:   "missing host",
			raw:    "http://",
			wantOK: false,
		},
		{
			name:   "bare path without host",
			raw:    "/just/a/path",
			wantOK: false,
		},
		{
			name:   "whitespace only after trim handled by caller",
			raw:    "   ",
			wantOK: false,
		},
		{
			name:   "plain text without host",
			raw:    "not a url at all",
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, ok := parseURL(tt.raw)
			if ok != tt.wantOK {
				t.Fatalf("parseURL(%q) ok=%v, want %v", tt.raw, ok, tt.wantOK)
			}
			if !tt.wantOK {
				return
			}
			if got := u.String(); got != tt.wantStr {
				t.Fatalf("parseURL(%q).String() = %q, want %q", tt.raw, got, tt.wantStr)
			}
		})
	}
}

func TestStripDefaultPortTable(t *testing.T) {
	tests := []struct {
		name   string
		scheme string
		host   string
		want   string
	}{
		{name: "http strips 80", scheme: "http", host: "host:80", want: "host"},
		{name: "https strips 443", scheme: "https", host: "host:443", want: "host"},
		{name: "http keeps 8080", scheme: "http", host: "host:8080", want: "host:8080"},
		{name: "https keeps 8443", scheme: "https", host: "host:8443", want: "host:8443"},
		{name: "ipv6 https strips 443", scheme: "https", host: "[::1]:443", want: "[::1]"},
		{name: "ipv6 http keeps non-default", scheme: "http", host: "[::1]:8080", want: "[::1]:8080"},
		{name: "other scheme unchanged", scheme: "ftp", host: "host:21", want: "host:21"},
		{name: "no port unchanged", scheme: "https", host: "example.com", want: "example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := stripDefaultPort(tt.scheme, tt.host); got != tt.want {
				t.Fatalf("stripDefaultPort(%q, %q) = %q, want %q", tt.scheme, tt.host, got, tt.want)
			}
		})
	}
}

func TestCanonicalizeTable(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "lowercase scheme host strip default port trim path drop fragment",
			in:   "HTTPS://Example.COM:443/a/b/?q=1#frag",
			want: "https://example.com/a/b?q=1",
		},
		{
			name: "empty path becomes slash",
			in:   "https://a.test",
			want: "https://a.test/",
		},
		{
			name: "root path stays slash",
			in:   "https://a.test/",
			want: "https://a.test/",
		},
		{
			name: "http default port stripped",
			in:   "http://HOST:80/x/",
			want: "http://host/x",
		},
		{
			name: "scheme relative gets https default",
			in:   "//HOST/a/",
			want: "https://host/a",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, ok := parseURL(tt.in)
			if !ok {
				t.Fatalf("parseURL(%q) ok=false", tt.in)
			}
			canonicalize(u)
			if got := u.String(); got != tt.want {
				t.Fatalf("canonicalize(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestHostKeyTable(t *testing.T) {
	tests := []struct {
		name      string
		raw       string
		normalize bool
		want      string
	}{
		{
			name:      "normalize lowers and strips default port",
			raw:       "https://Example.COM:443/path",
			normalize: true,
			want:      "example.com",
		},
		{
			name:      "without normalize preserves case",
			raw:       "https://Example.COM:443/path",
			normalize: false,
			want:      "Example.COM:443",
		},
		{
			name:      "non-default port kept when normalized",
			raw:       "http://HOST:8080/x",
			normalize: true,
			want:      "host:8080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, ok := parseURL(tt.raw)
			if !ok {
				t.Fatalf("parseURL(%q) ok=false", tt.raw)
			}
			if tt.normalize {
				canonicalize(u)
			}
			if got := hostKey(u, tt.normalize); got != tt.want {
				t.Fatalf("hostKey(%q, normalize=%v) = %q, want %q", tt.raw, tt.normalize, got, tt.want)
			}
		})
	}
}

func TestStripDefaultPortIdempotentProperty(t *testing.T) {
	schemes := []string{"http", "https", "ftp", "ws"}

	prop := func(schemeIdx uint8, host string) bool {
		if host == "" {
			return true
		}
		scheme := schemes[int(schemeIdx)%len(schemes)]
		once := stripDefaultPort(scheme, host)
		twice := stripDefaultPort(scheme, once)
		return once == twice
	}

	if err := quick.Check(prop, normalizeQuick); err != nil {
		t.Error(err)
	}
}

func TestCanonicalizeIdempotentProperty(t *testing.T) {
	paths := []string{"", "/", "/a", "/a/b", "/a/b/"}

	prop := func(pathIdx uint8, useHTTP bool) bool {
		scheme := "https"
		port := ":443"
		if useHTTP {
			scheme = "http"
			port = ":80"
		}
		raw := scheme + "://Example.COM" + port + paths[int(pathIdx)%len(paths)] + "#frag"
		u, ok := parseURL(raw)
		if !ok {
			return false
		}
		canonicalize(u)
		first := u.String()

		u2, err := url.Parse(first)
		if err != nil {
			return false
		}
		canonicalize(u2)
		return u2.String() == first
	}

	if err := quick.Check(prop, normalizeQuick); err != nil {
		t.Error(err)
	}
}
