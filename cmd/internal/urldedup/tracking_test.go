package urldedup

import (
	"net/url"
	"testing"
)

func TestStripTrackingParams(t *testing.T) {
	u, err := url.Parse("https://a.test/page?utm_source=x&page=2&fbclid=abc")
	if err != nil {
		t.Fatal(err)
	}
	stripTrackingParams(u)
	if u.RawQuery != "page=2" {
		t.Fatalf("RawQuery = %q, want page=2", u.RawQuery)
	}
}

func TestStripTrackingParamsAllRemoved(t *testing.T) {
	u, err := url.Parse("https://a.test/?utm_medium=cpc&gclid=1")
	if err != nil {
		t.Fatal(err)
	}
	stripTrackingParams(u)
	if u.RawQuery != "" {
		t.Fatalf("RawQuery = %q, want empty", u.RawQuery)
	}
}

func TestStripTrackingParamsCaseInsensitive(t *testing.T) {
	u, err := url.Parse("https://a.test/?UTM_SOURCE=x&keep=y")
	if err != nil {
		t.Fatal(err)
	}
	stripTrackingParams(u)
	if u.RawQuery != "keep=y" {
		t.Fatalf("RawQuery = %q, want keep=y", u.RawQuery)
	}
}

func TestStripTrackingParamsInvalidQuery(t *testing.T) {
	u, err := url.Parse("https://a.test/?bad=%ZZ")
	if err != nil {
		t.Fatal(err)
	}
	raw := u.RawQuery
	stripTrackingParams(u)
	if u.RawQuery != raw {
		t.Fatalf("invalid query should be left unchanged: got %q want %q", u.RawQuery, raw)
	}
}
