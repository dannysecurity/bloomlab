package bloom

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// KeyDistribution selects synthetic key shapes for hash tuning probes.
// Different workloads (URLs, fixed-width IDs, sequential counters) can favor
// different hash strategies; tune against keys that resemble production data.
type KeyDistribution int

const (
	// KeySequential uses "{prefix}-{i}" keys (default hashtune behavior).
	KeySequential KeyDistribution = iota
	// KeyURL uses HTTPS URLs with hosts, paths, and query parameters.
	KeyURL
	// KeyUUID uses deterministic UUID-shaped strings.
	KeyUUID
	// KeyFixed32 uses 32-byte binary keys with the index encoded in the tail.
	KeyFixed32
	// KeyFromSamples cycles through caller-provided sample keys.
	KeyFromSamples
)

// String returns the CLI-friendly distribution name.
func (d KeyDistribution) String() string {
	switch d {
	case KeySequential:
		return "sequential"
	case KeyURL:
		return "url"
	case KeyUUID:
		return "uuid"
	case KeyFixed32:
		return "fixed32"
	case KeyFromSamples:
		return "samples"
	default:
		return fmt.Sprintf("distribution(%d)", int(d))
	}
}

// ParseKeyDistribution maps a name to a KeyDistribution. Names are case-insensitive.
func ParseKeyDistribution(name string) (KeyDistribution, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", "sequential", "seq":
		return KeySequential, nil
	case "url", "urls":
		return KeyURL, nil
	case "uuid", "uuids":
		return KeyUUID, nil
	case "fixed32", "fixed", "binary32":
		return KeyFixed32, nil
	case "samples", "sample", "file":
		return KeyFromSamples, nil
	default:
		return 0, fmt.Errorf("bloom: unknown key distribution %q (want sequential, url, uuid, fixed32, or samples)", name)
	}
}

// KeyForDistribution returns a probe key generator for the given distribution.
func KeyForDistribution(dist KeyDistribution, prefix string) func(i int) []byte {
	switch dist {
	case KeyURL:
		return urlKeyFor(prefix)
	case KeyUUID:
		return uuidKeyFor(prefix)
	case KeyFixed32:
		return fixed32KeyFor(prefix)
	default:
		return sequentialKeyFor(prefix)
	}
}

// KeyForSamples returns a probe generator that cycles through sample keys.
// Empty entries are skipped; if all keys are empty, every probe returns nil.
func KeyForSamples(keys [][]byte) func(i int) []byte {
	if len(keys) == 0 {
		return func(int) []byte { return nil }
	}
	return func(i int) []byte {
		return keys[i%len(keys)]
	}
}

// LoadKeyFile reads up to maxKeys non-empty lines from path as probe keys.
func LoadKeyFile(path string, maxKeys int) ([][]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("bloom: open key file %q: %w", path, err)
	}
	defer f.Close()

	if maxKeys <= 0 {
		maxKeys = 10_000
	}

	keys := make([][]byte, 0, maxKeys)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		keys = append(keys, []byte(line))
		if len(keys) >= maxKeys {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("bloom: read key file %q: %w", path, err)
	}
	if len(keys) == 0 {
		return nil, fmt.Errorf("bloom: key file %q has no non-empty lines", path)
	}
	return keys, nil
}

func sequentialKeyFor(prefix string) func(i int) []byte {
	if prefix == "" {
		prefix = "tune"
	}
	p := prefix
	return func(i int) []byte {
		return []byte(fmt.Sprintf("%s-%d", p, i))
	}
}

func urlKeyFor(prefix string) func(i int) []byte {
	hosts := []string{"example.com", "cdn.example.net", "api.service.io", "static.assets.co"}
	paths := []string{"page", "item", "user", "article", "product"}
	if prefix == "" {
		prefix = "ref"
	}
	p := prefix
	return func(i int) []byte {
		host := hosts[i%len(hosts)]
		path := paths[(i/len(hosts))%len(paths)]
		return []byte(fmt.Sprintf("https://%s/%s/%s-%d?utm=%s&idx=%d", host, path, p, i, p, i))
	}
}

func uuidKeyFor(prefix string) func(i int) []byte {
	var pre uint32
	for _, b := range []byte(prefix) {
		pre = pre*31 + uint32(b)
	}
	return func(i int) []byte {
		n := uint64(i)*0x9e3779b97f4a7c15 + uint64(pre)
		return []byte(fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
			uint32(n>>32), uint16(n>>16), uint16(n), uint16(n>>8), n&0xffffffffffff))
	}
}

func fixed32KeyFor(prefix string) func(i int) []byte {
	var pre [32]byte
	copy(pre[:], prefix)
	return func(i int) []byte {
		key := make([]byte, 32)
		copy(key, pre[:])
		key[24] = byte(i >> 24)
		key[25] = byte(i >> 16)
		key[26] = byte(i >> 8)
		key[27] = byte(i)
		key[28] = byte(i * 7)
		key[29] = byte(i * 13)
		key[30] = byte(i * 31)
		key[31] = byte(i * 37)
		return key
	}
}
