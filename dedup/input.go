package dedup

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// InputMode controls how positional CLI arguments are interpreted.
type InputMode int

const (
	// InputModeLines treats each argument as one input line.
	InputModeLines InputMode = iota
	// InputModeFile treats zero arguments as stdin and one argument as a file path.
	InputModeFile
)

// InputSource describes where stream dedup should read lines from.
type InputSource struct {
	Reader io.Reader
	Label  string
}

// OpenInput resolves stdin, joined argument lines, or an optional file path.
// When stdin is a terminal and no arguments are given, sampleNeeded is true so
// callers can substitute built-in demo data.
func OpenInput(mode InputMode, args []string) (src InputSource, sampleNeeded bool, closeFn func(), err error) {
	closeFn = func() {}

	switch mode {
	case InputModeLines:
		if len(args) > 0 {
			lines := strings.Join(args, "\n") + "\n"
			return InputSource{Reader: strings.NewReader(lines), Label: fmt.Sprintf("%d argument(s)", len(args))}, false, closeFn, nil
		}
	case InputModeFile:
		switch len(args) {
		case 0:
			// fall through to stdin
		case 1:
			f, err := os.Open(args[0])
			if err != nil {
				return InputSource{}, false, closeFn, err
			}
			closeFn = func() { _ = f.Close() }
			return InputSource{Reader: f, Label: args[0]}, false, closeFn, nil
		default:
			return InputSource{}, false, closeFn, fmt.Errorf("expected 0 or 1 file argument, got %d", len(args))
		}
	default:
		return InputSource{}, false, closeFn, fmt.Errorf("dedup: unknown input mode %d", mode)
	}

	fi, err := os.Stdin.Stat()
	if err != nil {
		return InputSource{}, false, closeFn, err
	}
	if fi.Mode()&os.ModeCharDevice != 0 {
		return InputSource{}, true, closeFn, nil
	}
	return InputSource{Reader: os.Stdin, Label: "stdin"}, false, closeFn, nil
}
