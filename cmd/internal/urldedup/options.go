package urldedup

// Options configures how stdin lines are mapped to dedup keys.
type Options struct {
	// Normalize canonicalizes parseable URLs (scheme/host case, default ports,
	// trailing slashes, fragments, and protocol-relative //host paths).
	Normalize bool
	// StripQuery removes the entire query string before deduplication.
	StripQuery bool
	// StripTracking removes common marketing/click-tracking query parameters
	// (utm_*, fbclid, gclid, etc.) while preserving other query keys.
	StripTracking bool
	// StripFragment removes the URL fragment (#...) before deduplication without
	// applying other canonicalization rules.
	StripFragment bool
	// DomainOnly deduplicates by host name only, ignoring path and query.
	DomainOnly bool
}
