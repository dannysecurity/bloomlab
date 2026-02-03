package streamdedup

import (
	"bytes"
	"strings"
	"testing"

	"github.com/dannysecurity/bloomlab/cmd/internal/urldedup"
)

func TestRunURLNormalizeDupOnly(t *testing.T) {
	d := New(testFilter(t), func(line string) (string, bool) {
		return urldedup.Key(line, urldedup.Options{Normalize: true})
	})
	var out, errOut bytes.Buffer
	in := strings.NewReader(
		"https://Example.com/\n" +
			"http://example.com:80\n" +
			"https://other.test\n" +
			"HTTPS://EXAMPLE.COM:443/\n",
	)

	if err := Run(d, in, RunOptions{DupOnly: true, Out: &out, ErrOut: &errOut}); err != nil {
		t.Fatal(err)
	}

	wantOut := "dup\tHTTPS://EXAMPLE.COM:443/\n"
	if out.String() != wantOut {
		t.Fatalf("stdout:\n%s\nwant:\n%s", out.String(), wantOut)
	}
	if !strings.Contains(errOut.String(), "novel: 3, duplicates: 1") {
		t.Fatalf("stderr summary: %q", errOut.String())
	}
}

func TestRunURLStripTrackingNovelOnly(t *testing.T) {
	d := New(testFilter(t), func(line string) (string, bool) {
		return urldedup.Key(line, urldedup.Options{Normalize: true, StripTracking: true})
	})
	var out bytes.Buffer
	in := strings.NewReader(
		"https://a.test/page?utm_source=email&id=1\n" +
			"https://a.test/page?fbclid=x&id=1\n" +
			"https://a.test/page?id=2\n",
	)

	if err := Run(d, in, RunOptions{NovelOnly: true, Out: &out, ErrOut: ioDiscard{}}); err != nil {
		t.Fatal(err)
	}

	wantOut := "new\thttps://a.test/page?utm_source=email&id=1\n" +
		"new\thttps://a.test/page?id=2\n"
	if out.String() != wantOut {
		t.Fatalf("stdout:\n%s\nwant:\n%s", out.String(), wantOut)
	}
}
