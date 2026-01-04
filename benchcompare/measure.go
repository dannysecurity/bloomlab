package benchcompare

import (
	"fmt"
	"runtime"
	"time"

	"github.com/dannysecurity/bloomlab/bloom"
)

// Compare runs every scenario in AllScenarios and returns paired results.
func Compare(cfg Config) ([]Comparison, error) {
	if err := cfg.Bloom.Validate(); err != nil {
		return nil, fmt.Errorf("benchcompare: %w", err)
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
	case ScenarioContainsMixed:
		return compareContainsMixed(cfg)
	case ScenarioMixedStream:
		return compareMixedStream(cfg)
	case ScenarioRemove:
		return compareRemove(cfg)
	default:
		return Comparison{}, fmt.Errorf("benchcompare: unknown scenario %q", sc)
	}
}

func compareAdd(cfg Config) (Comparison, error) {
	keys := makeKeys(cfg.Bloom.ExpectedCapacity)

	bloomStart := time.Now()
	f, err := bloom.NewFilter(cfg.Bloom)
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
	set := make(map[string]struct{}, int(cfg.Bloom.ExpectedCapacity))
	for _, key := range keys {
		set[string(key)] = struct{}{}
	}
	runtime.ReadMemStats(&after)
	hashElapsed := time.Since(hashStart)
	hashBytes := float64(after.HeapAlloc - before.HeapAlloc)

	n := len(keys)
	bloomAllocs := allocsPerOp(n, func() {
		ff, err := bloom.NewFilter(cfg.Bloom)
		if err != nil {
			panic(err)
		}
		for _, key := range keys {
			ff.Add(key)
		}
	})
	hashAllocs := allocsPerOp(n, func() {
		s := make(map[string]struct{}, int(cfg.Bloom.ExpectedCapacity))
		for _, key := range keys {
			s[string(key)] = struct{}{}
		}
	})

	nf := float64(n)
	return Comparison{
		Scenario: ScenarioAdd,
		Bloom: BackendResult{
			NsPerOp:      float64(bloomElapsed.Nanoseconds()) / nf,
			BytesPerItem: bloomBytes / nf,
			AllocsPerOp:  bloomAllocs,
			TheoryFPR:    f.TheoryFPR(),
		},
		HashSet: BackendResult{
			NsPerOp:      float64(hashElapsed.Nanoseconds()) / nf,
			BytesPerItem: hashBytes / nf,
			AllocsPerOp:  hashAllocs,
		},
	}, nil
}

func compareContainsHit(cfg Config) (Comparison, error) {
	keys := makeKeys(cfg.Bloom.ExpectedCapacity)
	f, err := bloom.NewFilter(cfg.Bloom)
	if err != nil {
		return Comparison{}, err
	}
	set := make(map[string]struct{}, int(cfg.Bloom.ExpectedCapacity))
	for _, key := range keys {
		f.Add(key)
		set[string(key)] = struct{}{}
	}

	repeats := cfg.LookupRepeats
	totalOps := len(keys) * repeats

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

	bloomAllocs := allocsPerOp(totalOps, func() {
		for r := 0; r < repeats; r++ {
			for _, key := range keys {
				_ = f.Contains(key)
			}
		}
	})
	hashAllocs := allocsPerOp(totalOps, func() {
		for r := 0; r < repeats; r++ {
			for _, key := range keys {
				_, _ = set[string(key)]
			}
		}
	})

	total := float64(totalOps)
	return Comparison{
		Scenario: ScenarioContainsHit,
		Bloom: BackendResult{
			NsPerOp:      float64(bloomElapsed.Nanoseconds()) / total,
			BytesPerItem: bloomBytes / float64(len(keys)),
			AllocsPerOp:  bloomAllocs,
			TheoryFPR:    f.TheoryFPR(),
		},
		HashSet: BackendResult{
			NsPerOp:      float64(hashElapsed.Nanoseconds()) / total,
			BytesPerItem: hashBytes / float64(len(keys)),
			AllocsPerOp:  hashAllocs,
		},
	}, nil
}

func compareContainsMiss(cfg Config) (Comparison, error) {
	seedKeys := makeKeys(cfg.Bloom.ExpectedCapacity)
	f, err := bloom.NewFilter(cfg.Bloom)
	if err != nil {
		return Comparison{}, err
	}
	set := make(map[string]struct{}, int(cfg.Bloom.ExpectedCapacity))
	for _, key := range seedKeys {
		f.Add(key)
		set[string(key)] = struct{}{}
	}

	missKeys := makeMissKeys(cfg.Bloom.ExpectedCapacity)
	repeats := cfg.LookupRepeats
	totalOps := len(missKeys) * repeats

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

	bloomAllocs := allocsPerOp(totalOps, func() {
		for r := 0; r < repeats; r++ {
			for _, key := range missKeys {
				_ = f.Contains(key)
			}
		}
	})
	hashAllocs := allocsPerOp(totalOps, func() {
		for r := 0; r < repeats; r++ {
			for _, key := range missKeys {
				_, _ = set[string(key)]
			}
		}
	})

	total := float64(totalOps)
	return Comparison{
		Scenario: ScenarioContainsMiss,
		Bloom: BackendResult{
			NsPerOp:      float64(bloomElapsed.Nanoseconds()) / total,
			BytesPerItem: bloomBytes / float64(len(seedKeys)),
			AllocsPerOp:  bloomAllocs,
			TheoryFPR:    f.TheoryFPR(),
		},
		HashSet: BackendResult{
			NsPerOp:      float64(hashElapsed.Nanoseconds()) / total,
			BytesPerItem: hashBytes / float64(len(seedKeys)),
			AllocsPerOp:  hashAllocs,
		},
	}, nil
}

func compareContainsMixed(cfg Config) (Comparison, error) {
	hitRatio := cfg.LookupHitRatio
	if hitRatio < 0 || hitRatio > 1 {
		return Comparison{}, fmt.Errorf("benchcompare: LookupHitRatio must be in [0, 1]")
	}

	seedKeys := makeKeys(cfg.Bloom.ExpectedCapacity)
	f, err := bloom.NewFilter(cfg.Bloom)
	if err != nil {
		return Comparison{}, err
	}
	set := make(map[string]struct{}, int(cfg.Bloom.ExpectedCapacity))
	for _, key := range seedKeys {
		f.Add(key)
		set[string(key)] = struct{}{}
	}

	lookupKeys := makeMixedLookupKeys(cfg.Bloom.ExpectedCapacity, hitRatio)
	repeats := cfg.LookupRepeats
	totalOps := len(lookupKeys) * repeats

	bloomStart := time.Now()
	for r := 0; r < repeats; r++ {
		for _, key := range lookupKeys {
			_ = f.Contains(key)
		}
	}
	bloomElapsed := time.Since(bloomStart)

	hashStart := time.Now()
	for r := 0; r < repeats; r++ {
		for _, key := range lookupKeys {
			_, _ = set[string(key)]
		}
	}
	hashElapsed := time.Since(hashStart)

	bloomBytes := float64((f.BitCount() + 7) / 8)
	hashBytes := mapHeapBytes(set)

	bloomAllocs := allocsPerOp(totalOps, func() {
		for r := 0; r < repeats; r++ {
			for _, key := range lookupKeys {
				_ = f.Contains(key)
			}
		}
	})
	hashAllocs := allocsPerOp(totalOps, func() {
		for r := 0; r < repeats; r++ {
			for _, key := range lookupKeys {
				_, _ = set[string(key)]
			}
		}
	})

	total := float64(totalOps)
	return Comparison{
		Scenario:       ScenarioContainsMixed,
		LookupHitRatio: hitRatio,
		Bloom: BackendResult{
			NsPerOp:      float64(bloomElapsed.Nanoseconds()) / total,
			BytesPerItem: bloomBytes / float64(len(seedKeys)),
			AllocsPerOp:  bloomAllocs,
			TheoryFPR:    f.TheoryFPR(),
		},
		HashSet: BackendResult{
			NsPerOp:      float64(hashElapsed.Nanoseconds()) / total,
			BytesPerItem: hashBytes / float64(len(seedKeys)),
			AllocsPerOp:  hashAllocs,
		},
	}, nil
}

func compareMixedStream(cfg Config) (Comparison, error) {
	// First half unique, second half repeats — typical dedup stream shape.
	stream := makeMixedStream(cfg.Bloom.ExpectedCapacity)

	f, err := bloom.NewFilter(cfg.Bloom)
	if err != nil {
		return Comparison{}, err
	}
	set := make(map[string]struct{}, int(cfg.Bloom.ExpectedCapacity))

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

	n := len(stream)
	bloomBytes := float64((f.BitCount() + 7) / 8)
	hashBytes := mapHeapBytes(set)

	bloomAllocs := allocsPerOp(n, func() {
		ff, err := bloom.NewFilter(cfg.Bloom)
		if err != nil {
			panic(err)
		}
		for _, key := range stream {
			if ff.Contains(key) {
				continue
			}
			ff.Add(key)
		}
	})
	hashAllocs := allocsPerOp(n, func() {
		ss := make(map[string]struct{}, int(cfg.Bloom.ExpectedCapacity))
		for _, key := range stream {
			s := string(key)
			if _, ok := ss[s]; ok {
				continue
			}
			ss[s] = struct{}{}
		}
	})

	nf := float64(n)
	return Comparison{
		Scenario: ScenarioMixedStream,
		Bloom: BackendResult{
			NsPerOp:        float64(bloomElapsed.Nanoseconds()) / nf,
			BytesPerItem:   bloomBytes / float64(f.ApproximateCount()),
			AllocsPerOp:    bloomAllocs,
			TheoryFPR:      f.TheoryFPR(),
			FalsePositives: bloomFP,
		},
		HashSet: BackendResult{
			NsPerOp:        float64(hashElapsed.Nanoseconds()) / nf,
			BytesPerItem:   hashBytes / float64(len(set)),
			AllocsPerOp:    hashAllocs,
			FalsePositives: hashFP,
		},
	}, nil
}

func compareRemove(cfg Config) (Comparison, error) {
	keys := makeKeys(cfg.Bloom.ExpectedCapacity)

	cf, err := bloom.NewCountingFilter(cfg.Bloom)
	if err != nil {
		return Comparison{}, err
	}
	set := make(map[string]struct{}, int(cfg.Bloom.ExpectedCapacity))
	for _, key := range keys {
		if err := cf.Add(key); err != nil {
			return Comparison{}, err
		}
		set[string(key)] = struct{}{}
	}

	bloomBytes := float64(cf.CounterBytes())
	hashBytes := mapHeapBytes(set)

	bloomStart := time.Now()
	for _, key := range keys {
		cf.Remove(key)
	}
	bloomElapsed := time.Since(bloomStart)

	hashStart := time.Now()
	for _, key := range keys {
		delete(set, string(key))
	}
	hashElapsed := time.Since(hashStart)

	n := len(keys)

	bloomAllocs := allocsPerOp(n, func() {
		ff, err := bloom.NewCountingFilter(cfg.Bloom)
		if err != nil {
			panic(err)
		}
		for _, key := range keys {
			if err := ff.Add(key); err != nil {
				panic(err)
			}
		}
		for _, key := range keys {
			ff.Remove(key)
		}
	})
	hashAllocs := allocsPerOp(n, func() {
		ss := make(map[string]struct{}, int(cfg.Bloom.ExpectedCapacity))
		for _, key := range keys {
			ss[string(key)] = struct{}{}
		}
		for _, key := range keys {
			delete(ss, string(key))
		}
	})

	nf := float64(n)
	return Comparison{
		Scenario: ScenarioRemove,
		Bloom: BackendResult{
			NsPerOp:      float64(bloomElapsed.Nanoseconds()) / nf,
			BytesPerItem: bloomBytes / nf,
			AllocsPerOp:  bloomAllocs,
			TheoryFPR:    cf.TheoryFPR(),
		},
		HashSet: BackendResult{
			NsPerOp:      float64(hashElapsed.Nanoseconds()) / nf,
			BytesPerItem: hashBytes / nf,
			AllocsPerOp:  hashAllocs,
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

// makeMixedLookupKeys builds n lookup keys with hitRatio fraction present in the
// seeded set and the remainder absent. Keys are interleaved so neither backend
// sees long runs of hits or misses.
func makeMixedLookupKeys(n uint64, hitRatio float64) [][]byte {
	hitCount := int(float64(n) * hitRatio)
	if hitCount > int(n) {
		hitCount = int(n)
	}
	missCount := int(n) - hitCount
	keys := make([][]byte, int(n))
	hitIdx, missIdx := 0, 0
	for i := 0; i < int(n); i++ {
		targetHits := (i + 1) * hitCount / int(n)
		if hitIdx < targetHits {
			keys[i] = []byte(fmt.Sprintf("key-%d", hitIdx))
			hitIdx++
			continue
		}
		offset := int(n) + 1_000_000 + missIdx
		keys[i] = []byte(fmt.Sprintf("miss-%d", offset))
		missIdx++
	}
	if missIdx != missCount || hitIdx != hitCount {
		panic(fmt.Sprintf("benchcompare: mixed lookup key counts hit=%d miss=%d want hit=%d miss=%d",
			hitIdx, missIdx, hitCount, missCount))
	}
	return keys
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
