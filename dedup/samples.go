package dedup

import (
	"fmt"
	"io"
	"strings"
)

// SampleKind selects a built-in dedup demo dataset.
type SampleKind string

const (
	SampleStream   SampleKind = "stream"
	SampleURL      SampleKind = "url"
	SampleTracking SampleKind = "tracking"
)

// SampleLines returns the canonical lines for a built-in demo dataset.
func SampleLines(kind SampleKind) ([]string, error) {
	switch kind {
	case SampleStream:
		return []string{"alpha", "beta", "alpha", "gamma", "beta", "alpha"}, nil
	case SampleURL:
		return []string{
			"https://Example.com/page",
			"http://example.com:80/page",
			"https://other.test/path",
			"HTTPS://EXAMPLE.COM:443/page",
		}, nil
	case SampleTracking:
		return []string{
			"https://shop.test/item?utm_source=email&id=42",
			"https://shop.test/item?fbclid=abc&id=42",
			"https://shop.test/item?id=99",
			"https://shop.test/item?id=42",
		}, nil
	default:
		return nil, fmt.Errorf("unknown sample %q (want stream, url, or tracking)", kind)
	}
}

// ParseSampleKind normalizes a sample flag value.
func ParseSampleKind(raw string) (SampleKind, error) {
	switch SampleKind(raw) {
	case SampleStream, SampleURL, SampleTracking:
		return SampleKind(raw), nil
	default:
		return "", fmt.Errorf("unknown sample %q (want stream, url, or tracking)", raw)
	}
}

// SampleReader returns a reader over a built-in demo dataset.
func SampleReader(kind SampleKind) (io.Reader, error) {
	lines, err := SampleLines(kind)
	if err != nil {
		return nil, err
	}
	return strings.NewReader(strings.Join(lines, "\n") + "\n"), nil
}

// DefaultSampleKind picks a built-in dataset for interactive demos.
func DefaultSampleKind(urlMode bool) SampleKind {
	if urlMode {
		return SampleURL
	}
	return SampleStream
}
