package main

import (
	"net/url"
	"strings"
)

// canonicalKey returns the dedup key for a stdin line.
// Blank lines yield ok=false. When normalize is false the trimmed line is used as-is.
// When normalize is true, parseable URLs are canonicalized (scheme/host case,
// default ports, trailing slashes, fragments, and protocol-relative //host paths
// defaulting to https); other lines keep their trimmed form.
func canonicalKey(line string, normalize bool) (key string, ok bool) {
	line = strings.TrimSpace(line)
	if line == "" {
		return "", false
	}
	if !normalize {
		return line, true
	}
	return normalizeURL(line), true
}

func normalizeURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	if u.Scheme == "" && u.Host == "" && !strings.Contains(raw, "://") {
		if withScheme, err := url.Parse("https://" + raw); err == nil && withScheme.Host != "" {
			u = withScheme
		} else {
			return raw
		}
	}
	if u.Host == "" {
		return raw
	}

	u.Scheme = strings.ToLower(u.Scheme)
	if u.Scheme == "" {
		u.Scheme = "https"
	}
	u.Host = strings.ToLower(u.Host)
	u.Fragment = ""

	u.Host = stripDefaultPort(u.Scheme, u.Host)

	if u.Path == "" {
		u.Path = "/"
	} else if u.Path != "/" {
		u.Path = strings.TrimSuffix(u.Path, "/")
	}

	return u.String()
}

func stripDefaultPort(scheme, host string) string {
	switch scheme {
	case "http":
		if strings.HasSuffix(host, ":80") {
			return host[:len(host)-3]
		}
	case "https":
		if strings.HasSuffix(host, ":443") {
			return host[:len(host)-4]
		}
	}
	return host
}
