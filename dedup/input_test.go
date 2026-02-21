package dedup

import (
	"io"
	"strings"
	"testing"
)

func TestOpenInputLinesFromArgs(t *testing.T) {
	src, sample, closeFn, err := OpenInput(InputModeLines, []string{"a", "b"})
	if err != nil {
		t.Fatal(err)
	}
	defer closeFn()

	if sample {
		t.Fatal("args should not request a sample")
	}
	if src.Label != "2 argument(s)" {
		t.Fatalf("label = %q", src.Label)
	}

	b, err := io.ReadAll(src.Reader)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(b); got != "a\nb\n" {
		t.Fatalf("reader = %q, want %q", got, "a\nb\n")
	}
}

func TestOpenInputFileModeRejectsMultipleArgs(t *testing.T) {
	_, _, _, err := OpenInput(InputModeFile, []string{"a.txt", "b.txt"})
	if err == nil {
		t.Fatal("expected error for multiple file arguments")
	}
	if !strings.Contains(err.Error(), "expected 0 or 1 file argument") {
		t.Fatalf("error = %v", err)
	}
}

func TestSampleLines(t *testing.T) {
	for _, kind := range []SampleKind{SampleStream, SampleURL, SampleTracking} {
		lines, err := SampleLines(kind)
		if err != nil {
			t.Fatalf("%s: %v", kind, err)
		}
		if len(lines) < 3 {
			t.Fatalf("%s: got %d lines, want at least 3", kind, len(lines))
		}
	}

	if _, err := SampleLines(SampleKind("nope")); err == nil {
		t.Fatal("expected error for unknown sample")
	}
}

func TestParseSampleKind(t *testing.T) {
	kind, err := ParseSampleKind("tracking")
	if err != nil || kind != SampleTracking {
		t.Fatalf("kind=%q err=%v", kind, err)
	}

	if _, err := ParseSampleKind("nope"); err == nil {
		t.Fatal("expected error")
	}
}

func TestDefaultSampleKind(t *testing.T) {
	if got := DefaultSampleKind(false); got != SampleStream {
		t.Fatalf("stream mode default = %q", got)
	}
	if got := DefaultSampleKind(true); got != SampleURL {
		t.Fatalf("url mode default = %q", got)
	}
}

func TestSampleReader(t *testing.T) {
	r, err := SampleReader(SampleStream)
	if err != nil {
		t.Fatal(err)
	}
	b, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "alpha") {
		t.Fatalf("sample reader = %q", string(b))
	}
}
