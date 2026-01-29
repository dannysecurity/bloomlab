package urldedup

import "strings"

// Key maps a stdin line to a dedup key. Blank lines yield ok=false.
func Key(line string, opts Options) (key string, ok bool) {
	line = strings.TrimSpace(line)
	if line == "" {
		return "", false
	}

	needParse := opts.Normalize || opts.StripQuery || opts.StripTracking || opts.StripFragment || opts.DomainOnly
	if !needParse {
		return line, true
	}

	u, parsed := parseURL(line)
	if !parsed {
		if opts.DomainOnly {
			return "", false
		}
		return line, true
	}

	if opts.Normalize {
		canonicalize(u)
	}

	if opts.StripTracking {
		stripTrackingParams(u)
	}

	if opts.StripQuery {
		u.RawQuery = ""
		u.ForceQuery = false
	}

	if opts.StripFragment {
		u.Fragment = ""
	}

	if opts.DomainOnly {
		return hostKey(u, opts.Normalize), true
	}

	return u.String(), true
}
