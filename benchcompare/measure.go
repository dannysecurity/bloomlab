package benchcompare

import (
	"fmt"
	"runtime"
	"time"

	"github.com/dannysecurity/bloomlab/bloom"
)

// Compare runs every scenario in AllScenarios and returns paired results.
func Compare(cfg Config) ([]Comparison, error) {
	if cfg.ItemCount == 0 {
		return nil, fmt.Errorf("benchcompare: ItemCount must be > 0")
	}
	if cfg.FalsePositiveRate <= 0 || cfg.FalsePositiveRate >= 1 {
		return nil, fmt.Errorf("benchcompare: FalsePositiveRate must be in (0, 1)")
	}
	if cfg.LookupRepeats <= 0 {
		cfg.LookupRepeats = 1
	}

	out := make([]Comparison, 0, len(AllScenarios))
	for _, sc := range AllScenarios {
		cmp, err := compareScenario(cfg, sc)
		if err != nil {
			return nil, err
		}
		out = append(out, cmp)
	}
	return out, nil
}

func compareScenario(cfg Config, sc Scenario) (Comparison, error) {
	switch sc {
	case ScenarioAdd:
		return compareAdd(cfg)
	case ScenarioContainsHit:
		return compareContainsHit(cfg)
	case ScenarioContainsMiss:
		return compareContainsMiss(cfg)
	case ScenarioMixedStream:
		return compareMixedStream(cfg)
	default:
		return Comparison{}, fmt.Errorf("benchcompare: unknown scenario %q", sc)
	}
}

func compareAdd(cfg Config) (Comparison, error) {
	keys := makeKeys(cfg.ItemCount)

	bloomStart := time.Now()
	f, err := bloom.NewFilter(bloom.TargetConfig(cfg.ItemCount, cfg.FalsePositiveRate))
	if err != nil {
		return Comparison{}, err
	}
	for _, key := range keys {
		f.Add(key)
	}
	bloomElapsed := time.Since(bloomStart)
	bloomBytes := float64((f.BitCount() + 7) / 8)

	hashStart := time.Now()
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	set := make(map[string]struct{}, int(cfg.ItemCount))
	for _, key := range keys {
		set[string(key)] = struct{}{}
	}
	runtime.ReadMemStats(&after)
	hashElapsed := time.Since(hashStart)
	hashBytes := float64(after.HeapAlloc - before.HeapAlloc)

	n := float64(len(keys))
	return Comparison{
		Scenario: ScenarioAdd,
		Bloom: BackendResult{
			NsPerOp:      float64(bloomElapsed.Nanoseconds()) / n,
			BytesPerItem: bloomBytes / n,
			TheoryFPR:    f.TheoryFPR(),
		},
		HashSet: BackendResult{
			NsPerOp:      float64(hashElapsed.Nanoseconds()) / n,
			BytesPerItem: hashBytes / n,
		},
	}, nil
}

func compareContainsHit(cfg Config) (Comparison, error) {
	keys := makeKeys(cfg.ItemCount)
	f, err := bloom.NewFilter(bloom.TargetConfig(cfg.ItemCount, cfg.FalsePositiveRate))
	if err != nil {
		return Comparison{}, err
	}
	set := make(map[string]struct{}, int(cfg.ItemCount))
	for _, key := range keys {
		f.Add(key)
		set[string(key)] = struct{}{}
	}

	repeats := cfg.LookupRepeats
	totalOps := float64(len(keys) * repeats)

	bloomStart := time.Now()
	for r := 0; r < repeats; r++ {
		for _, key := range keys {
			_ = f.Contains(key)
		}
	}
	bloomElapsed := time.Since(bloomStart)

	hashStart := time.Now()
	for r := 0; r < repeats; r++ {
		for _, key := range keys {
			_, _ = set[string(key)]
		}
	}
	hashElapsed := time.Since(hashStart)

	bloomBytes := float64((f.BitCount() + 7) / 8)
	hashBytes := mapHeapBytes(set)

	return Comparison{
		Scenario: ScenarioContainsHit,
		Bloom: BackendResult{
			NsPerOp:      float64(bloomElapsed.Nanoseconds()) / totalOps,
			BytesPerItem: bloomBytes / float64(len(keys)),
			TheoryFPR:    f.TheoryFPR(),
		},
		HashSet: BackendResult{
			NsPerOp:      float64(hashElapsed.Nanoseconds()) / totalOps,
			BytesPerItem: hashBytes / float64(len(keys)),
		},
	}, nil
}

func compareContainsMiss(cfg Config) (Comparison, error) {
	seedKeys := makeKeys(cfg.ItemCount)
	f, err := bloom.NewFilter(bloom.TargetConfig(cfg.ItemCount, cfg.FalsePositiveRate))
	if err != nil {
		return Comparison{}, err
	}
	set := make(map[string]struct{}, int(cfg.ItemCount))
	for _, key := range seedKeys {
		f.Add(key)
		set[string(key)] = struct{}{}
	}

	missKeys := makeMissKeys(cfg.ItemCount)
	repeats := cfg.LookupRepeats
	totalOps := float64(len(missKeys) * repeats)

	bloomStart := time.Now()
	for r := 0; r < repeats; r++ {
		for _, key := range missKeys {
			_ = f.Contains(key)
		}
	}
	bloomElapsed := time.Since(bloomStart)

	hashStart := time.Now()
	for r := 0; r < repeats; r++ {
		for _, key := range missKeys {
			_, _ = set[string(key)]
		}
	}
	hashElapsed := time.Since(hashStart)

	bloomBytes := float64((f.BitCount() + 7) / 8)
	hashBytes := mapHeapBytes(set)

	return Comparison{
		Scenario: ScenarioContainsMiss,
		Bloom: BackendResult{
			NsPerOp:      float64(bloomElapsed.Nanoseconds()) / totalOps,
			BytesPerItem: bloomBytes / float64(len(seedKeys)),
			TheoryFPR:    f.TheoryFPR(),
		},
		HashSet: BackendResult{
			NsPerOp:      float64(hashElapsed.Nanoseconds()) / totalOps,
			BytesPerItem: hashBytes / float64(len(seedKeys)),
		},
	}, nil
}

func compareMixedStream(cfg Config) (Comparison, error) {
	// First half unique, second half repeats — typical dedup stream shape.
	stream := makeMixedStream(cfg.ItemCount)

	f, err := bloom.NewFilter(bloom.TargetConfig(cfg.ItemCount, cfg.FalsePositiveRate))
	if err != nil {
		return Comparison{}, err
	}
	set := make(map[string]struct{}, int(cfg.ItemCount))

	var bloomFP int
	bloomStart := time.Now()
	for _, key := range stream {
		if f.Contains(key) {
			bloomFP++
			continue
		}
		f.Add(key)
	}
	bloomElapsed := time.Since(bloomStart)

	var hashFP int
	hashStart := time.Now()
	for _, key := range stream {
		s := string(key)
		if _, ok := set[s]; ok {
			hashFP++
			continue
		}
		set[s] = struct{}{}
	}
	hashElapsed := time.Since(hashStart)

	n := float64(len(stream))
	bloomBytes := float64((f.BitCount() + 7) / 8)
	hashBytes := mapHeapBytes(set)

	return Comparison{
		Scenario: ScenarioMixedStream,
		Bloom: BackendResult{
			NsPerOp:        float64(bloomElapsed.Nanoseconds()) / n,
			BytesPerItem:   bloomBytes / float64(f.ApproximateCount()),
			TheoryFPR:      f.TheoryFPR(),
			FalsePositives: bloomFP,
		},
		HashSet: BackendResult{
			NsPerOp:        float64(hashElapsed.Nanoseconds()) / n,
			BytesPerItem:   hashBytes / float64(len(set)),
			FalsePositives: hashFP,
		},
	}, nil
}

func makeKeys(n uint64) [][]byte {
	keys := make([][]byte, n)
	for i := range keys {
		keys[i] = []byte(fmt.Sprintf("key-%d", i))
	}
	return keys
}

func makeMissKeys(n uint64) [][]byte {
	keys := make([][]byte, n)
	offset := int(n) + 1_000_000
	for i := range keys {
		keys[i] = []byte(fmt.Sprintf("miss-%d", i+offset))
	}
	return keys
}

func makeMixedStream(n uint64) [][]byte {
	half := int(n / 2)
	stream := make([][]byte, n)
	for i := 0; i < half; i++ {
		stream[i] = []byte(fmt.Sprintf("stream-%d", i))
	}
	for i := half; i < int(n); i++ {
		stream[i] = []byte(fmt.Sprintf("stream-%d", i%half))
	}
	return stream
}

func mapHeapBytes(set map[string]struct{}) float64 {
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	clone := make(map[string]struct{}, len(set))
	for k := range set {
		clone[k] = struct{}{}
	}
	runtime.ReadMemStats(&after)
	return float64(after.HeapAlloc - before.HeapAlloc)
}
