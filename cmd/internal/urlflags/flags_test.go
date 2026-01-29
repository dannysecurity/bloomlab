package urlflags

import (
	"flag"
	"testing"
)

func TestOptionsDefaults(t *testing.T) {
	flag.CommandLine = flag.NewFlagSet("test", flag.ContinueOnError)
	f := Register()
	if err := flag.CommandLine.Parse(nil); err != nil {
		t.Fatal(err)
	}

	opts := f.Options()
	if opts.Normalize || opts.StripQuery || opts.StripTracking || opts.StripFragment || opts.DomainOnly {
		t.Fatalf("Options() = %+v, want all false", opts)
	}
	key, ok := f.KeyFunc()("https://example.com")
	if !ok || key != "https://example.com" {
		t.Fatalf("KeyFunc() = %q ok=%v, want https://example.com", key, ok)
	}
}

func TestOptionsStripFragment(t *testing.T) {
	flag.CommandLine = flag.NewFlagSet("test", flag.ContinueOnError)
	f := Register()
	if err := flag.CommandLine.Parse([]string{"-strip-fragment"}); err != nil {
		t.Fatal(err)
	}

	opts := f.Options()
	if !opts.StripFragment || opts.Normalize || opts.StripQuery || opts.StripTracking || opts.DomainOnly {
		t.Fatalf("Options() = %+v, want strip-fragment only", opts)
	}
	key, ok := f.KeyFunc()("https://a.test/doc#intro")
	if !ok || key != "https://a.test/doc" {
		t.Fatalf("KeyFunc() = %q ok=%v, want https://a.test/doc", key, ok)
	}
}

func TestOptionsNormalizeAndStripTracking(t *testing.T) {
	flag.CommandLine = flag.NewFlagSet("test", flag.ContinueOnError)
	f := Register()
	if err := flag.CommandLine.Parse([]string{"-normalize", "-strip-tracking"}); err != nil {
		t.Fatal(err)
	}

	opts := f.Options()
	if !opts.Normalize || !opts.StripTracking || opts.StripQuery || opts.DomainOnly {
		t.Fatalf("Options() = %+v, want normalize and strip-tracking", opts)
	}
	key, ok := f.KeyFunc()("https://a.test/p?utm_source=x&id=1")
	if !ok {
		t.Fatal("KeyFunc() ok=false")
	}
	want := "https://a.test/p?id=1"
	if key != want {
		t.Fatalf("KeyFunc() = %q, want %q", key, want)
	}
}
