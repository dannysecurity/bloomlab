package urldedup

import (
	"net/url"
	"strings"
)

// trackingParams are query keys commonly used for campaign attribution.
// Stripping them helps dedupe URLs that differ only by tracking tags.
var trackingParams = map[string]struct{}{
	"utm_source":   {},
	"utm_medium":   {},
	"utm_campaign": {},
	"utm_term":     {},
	"utm_content":  {},
	"utm_id":       {},
	"fbclid":       {},
	"gclid":        {},
	"gclsrc":       {},
	"dclid":        {},
	"msclkid":      {},
	"mc_cid":       {},
	"mc_eid":       {},
	"_ga":          {},
	"_gl":          {},
	"ref":          {},
	"source":       {},
}

func stripTrackingParams(u *url.URL) {
	if u.RawQuery == "" {
		return
	}
	values, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return
	}
	for key := range values {
		if _, drop := trackingParams[strings.ToLower(key)]; drop {
			delete(values, key)
		}
	}
	u.RawQuery = values.Encode()
	if u.RawQuery == "" {
		u.ForceQuery = false
	}
}
