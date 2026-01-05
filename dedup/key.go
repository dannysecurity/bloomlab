package dedup

import "strings"

// KeyFunc maps a line to a dedup key. Return ok=false to skip the line.
type KeyFunc func(line string) (key string, ok bool)

// TrimKey trims whitespace and skips blank lines.
func TrimKey(line string) (string, bool) {
	line = strings.TrimSpace(line)
	if line == "" {
		return "", false
	}
	return line, true
}

// IgnoreCaseKey trims whitespace, lowercases, and skips blank lines.
func IgnoreCaseKey(line string) (string, bool) {
	key, ok := TrimKey(line)
	if !ok {
		return "", false
	}
	return strings.ToLower(key), true
}
