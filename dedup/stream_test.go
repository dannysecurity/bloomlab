package dedup

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

func testCountingFilter(t *testing.T) *bloom.CountingFilter {
	t.Helper()
	f, err := bloom.NewCountingFilter(bloom.TargetConfig(100, 0.01))
	if err != nil {
		t.Fatal(err)
	}
	return f
}

func TestClassifierClassify(t *testing.T) {
	c := NewClassifier(testFilter(t), nil)

	dup, ok := c.Classify("  alpha  ")
	if !ok || dup {
		t.Fatalf("first line: ok=%v dup=%v, want ok=true dup=false", ok, dup)
	}

	dup, ok = c.Classify("alpha")
	if !ok || !dup {
		t.Fatalf("repeat line: ok=%v dup=%v, want ok=true dup=true", ok, dup)
	}

	dup, ok = c.Classify("beta")
	if !ok || dup {
		t.Fatalf("novel line: ok=%v dup=%v, want ok=true dup=false", ok, dup)
	}

	dup, ok = c.Classify("   ")
	if ok || dup {
		t.Fatalf("blank line: ok=%v dup=%v, want ok=false dup=false", ok, dup)
	}

	if c.Novel() != 2 || c.Duplicates() != 1 {
		t.Fatalf("counts: novel=%d dup=%d, want novel=2 dup=1", c.Novel(), c.Duplicates())
	}
}

func TestIgnoreCaseKey(t *testing.T) {
	c := NewClassifier(testFilter(t), IgnoreCaseKey)

	dup, ok := c.Classify("Hello")
	if !ok || dup {
		t.Fatalf("first visit: ok=%v dup=%v", ok, dup)
	}

	dup, ok = c.Classify("hello")
	if !ok || !dup {
		t.Fatalf("case-insensitive duplicate: ok=%v dup=%v", ok, dup)
	}
}

func TestCountingClassifierRemove(t *testing.T) {
	c := NewCountingClassifier(testCountingFilter(t), nil)

	dup, ok, err := c.Classify("alpha")
	if err != nil || !ok || dup {
		t.Fatalf("first insert: err=%v ok=%v dup=%v", err, ok, dup)
	}

	dup, ok, err = c.Classify("alpha")
	if err != nil || !ok || !dup {
		t.Fatalf("duplicate: err=%v ok=%v dup=%v", err, ok, dup)
	}

	removed, ok := c.Remove("alpha")
	if !removed || !ok {
		t.Fatalf("remove: removed=%v ok=%v", removed, ok)
	}

	dup, ok, err = c.Classify("alpha")
	if err != nil || !ok || dup {
		t.Fatalf("re-insert after remove: err=%v ok=%v dup=%v", err, ok, dup)
	}

	if c.Novel() != 2 || c.Duplicates() != 1 || c.Removed() != 1 {
		t.Fatalf("counts: novel=%d dup=%d removed=%d", c.Novel(), c.Duplicates(), c.Removed())
	}
}

func TestRunTextOutput(t *testing.T) {
	c := NewClassifier(testFilter(t), nil)
	var out, errOut bytes.Buffer
	in := strings.NewReader("a\nb\na\n\n")

	if err := Run(c, in, RunOptions{Out: &out, ErrOut: &errOut}); err != nil {
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

func TestRunCountingWithRemovePrefix(t *testing.T) {
	c := NewCountingClassifier(testCountingFilter(t), nil)
	var out, errOut bytes.Buffer
	in := strings.NewReader("a\na\n-a\na\n")

	if err := RunCounting(c, in, CountingRunOptions{
		RunOptions:   RunOptions{Out: &out, ErrOut: &errOut},
		RemovePrefix: "-",
	}); err != nil {
		t.Fatal(err)
	}

	wantOut := "new\ta\ndup\ta\nremoved\t-a\nnew\ta\n"
	if out.String() != wantOut {
		t.Fatalf("stdout:\n%s\nwant:\n%s", out.String(), wantOut)
	}
	if !strings.Contains(errOut.String(), "removed: 1") {
		t.Fatalf("stderr summary: %q", errOut.String())
	}
}

func TestRunDupOnly(t *testing.T) {
	c := NewClassifier(testFilter(t), nil)
	var out, errOut bytes.Buffer
	in := strings.NewReader("a\nb\na\nc\na\n")

	if err := Run(c, in, RunOptions{DupOnly: true, Out: &out, ErrOut: &errOut}); err != nil {
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

func TestRunCountingDupOnly(t *testing.T) {
	c := NewCountingClassifier(testCountingFilter(t), nil)
	var out, errOut bytes.Buffer
	in := strings.NewReader("a\nb\na\nc\na\n")

	if err := RunCounting(c, in, CountingRunOptions{
		RunOptions: RunOptions{DupOnly: true, Out: &out, ErrOut: &errOut},
	}); err != nil {
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

func TestRunJSONOutput(t *testing.T) {
	c := NewClassifier(testFilter(t), nil)
	var out bytes.Buffer
	in := strings.NewReader("x\nx\n")

	if err := Run(c, in, RunOptions{Format: FormatJSON, Out: &out, ErrOut: ioDiscard{}}); err != nil {
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

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) { return len(p), nil }
