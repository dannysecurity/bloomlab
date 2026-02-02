package streamdedup

import (
	"bytes"
	"strings"
	"testing"

	"github.com/dannysecurity/bloomlab/bloom"
)

func testFilter(t *testing.T) *bloom.Filter {
	t.Helper()
	f, err := bloom.NewFilter(bloom.TargetConfig(100, 0.01))
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func TestClassify(t *testing.T) {
	d := New(testFilter(t), nil)

	dup, ok := d.Classify("  alpha  ")
	if !ok || dup {
		t.Fatalf("first line: ok=%v dup=%v, want ok=true dup=false", ok, dup)
	}

	dup, ok = d.Classify("alpha")
	if !ok || !dup {
		t.Fatalf("repeat line: ok=%v dup=%v, want ok=true dup=true", ok, dup)
	}

	dup, ok = d.Classify("beta")
	if !ok || dup {
		t.Fatalf("novel line: ok=%v dup=%v, want ok=true dup=false", ok, dup)
	}

	dup, ok = d.Classify("   ")
	if ok || dup {
		t.Fatalf("blank line: ok=%v dup=%v, want ok=false dup=false", ok, dup)
	}

	if d.Novel() != 2 || d.Duplicates() != 1 {
		t.Fatalf("counts: novel=%d dup=%d, want novel=2 dup=1", d.Novel(), d.Duplicates())
	}
}

func TestIgnoreCaseKey(t *testing.T) {
	d := New(testFilter(t), IgnoreCaseKey)

	dup, ok := d.Classify("Hello")
	if !ok || dup {
		t.Fatalf("first visit: ok=%v dup=%v", ok, dup)
	}

	dup, ok = d.Classify("hello")
	if !ok || !dup {
		t.Fatalf("case-insensitive duplicate: ok=%v dup=%v", ok, dup)
	}
}

func TestRunTextOutput(t *testing.T) {
	d := New(testFilter(t), nil)
	var out, errOut bytes.Buffer
	in := strings.NewReader("a\nb\na\n\n")

	if err := Run(d, in, RunOptions{Out: &out, ErrOut: &errOut}); err != nil {
		t.Fatal(err)
	}

	wantOut := "new\ta\nnew\tb\ndup\ta\n"
	if out.String() != wantOut {
		t.Fatalf("stdout:\n%s\nwant:\n%s", out.String(), wantOut)
	}
	if !strings.Contains(errOut.String(), "novel: 2, duplicates: 1") {
		t.Fatalf("stderr summary: %q", errOut.String())
	}
}

func TestRunJSONOutput(t *testing.T) {
	d := New(testFilter(t), nil)
	var out bytes.Buffer
	in := strings.NewReader("x\nx\n")

	if err := Run(d, in, RunOptions{Format: FormatJSON, Out: &out, ErrOut: ioDiscard{}}); err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("got %d json lines, want 2", len(lines))
	}
	if lines[0] != `{"status":"new","line":"x"}` {
		t.Fatalf("first line: %s", lines[0])
	}
	if lines[1] != `{"status":"dup","line":"x"}` {
		t.Fatalf("second line: %s", lines[1])
	}
}

func TestRunQuiet(t *testing.T) {
	d := New(testFilter(t), nil)
	var out, errOut bytes.Buffer
	in := strings.NewReader("only\nonly\n")

	if err := Run(d, in, RunOptions{Quiet: true, Out: &out, ErrOut: &errOut}); err != nil {
		t.Fatal(err)
	}
	if out.String() != "" {
		t.Fatalf("quiet mode wrote stdout: %q", out.String())
	}
	if !strings.Contains(errOut.String(), "novel: 1, duplicates: 1") {
		t.Fatalf("stderr summary: %q", errOut.String())
	}
}

func TestRunNovelOnly(t *testing.T) {
	d := New(testFilter(t), nil)
	var out, errOut bytes.Buffer
	in := strings.NewReader("a\nb\na\nc\na\n")

	if err := Run(d, in, RunOptions{NovelOnly: true, Out: &out, ErrOut: &errOut}); err != nil {
		t.Fatal(err)
	}

	wantOut := "new\ta\nnew\tb\nnew\tc\n"
	if out.String() != wantOut {
		t.Fatalf("stdout:\n%s\nwant:\n%s", out.String(), wantOut)
	}
	if !strings.Contains(errOut.String(), "novel: 3, duplicates: 2") {
		t.Fatalf("stderr summary: %q", errOut.String())
	}
}

func TestRunDupOnly(t *testing.T) {
	d := New(testFilter(t), nil)
	var out, errOut bytes.Buffer
	in := strings.NewReader("a\nb\na\nc\na\n")

	if err := Run(d, in, RunOptions{DupOnly: true, Out: &out, ErrOut: &errOut}); err != nil {
		t.Fatal(err)
	}

	wantOut := "dup\ta\ndup\ta\n"
	if out.String() != wantOut {
		t.Fatalf("stdout:\n%s\nwant:\n%s", out.String(), wantOut)
	}
	if !strings.Contains(errOut.String(), "novel: 3, duplicates: 2") {
		t.Fatalf("stderr summary: %q", errOut.String())
	}
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) { return len(p), nil }
