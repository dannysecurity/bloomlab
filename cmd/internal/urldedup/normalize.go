package urldedup

import (
	"net/url"
	"strings"
)

func parseURL(raw string) (*url.URL, bool) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, false
	}
	if u.Scheme == "" && u.Host == "" && !strings.Contains(raw, "://") {
		if withScheme, err := url.Parse("https://" + raw); err == nil && withScheme.Host != "" {
			u = withScheme
		} else {
			return nil, false
		}
	}
	if u.Host == "" {
		return nil, false
	}
	return u, true
}

func canonicalize(u *url.URL) {
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

func hostKey(u *url.URL, normalize bool) string {
	host := u.Host
	if normalize {
		host = strings.ToLower(host)
		host = stripDefaultPort(u.Scheme, host)
	}
	return host
}
