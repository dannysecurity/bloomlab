package benchcompare

import (
	"strings"
	"testing"

	"github.com/dannysecurity/bloomlab/bloom"
)

func TestCompareAllScenarios(t *testing.T) {
	cfg := Config{Bloom: bloom.TargetFilter(2_000, 0.01),
		LookupRepeats:     2,
	}
	results, err := Compare(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != len(AllScenarios) {
		t.Fatalf("got %d results, want %d", len(results), len(AllScenarios))
	}
	for _, cmp := range results {
		if cmp.Bloom.NsPerOp <= 0 {
			t.Errorf("%s: bloom ns/op must be positive", cmp.Scenario)
		}
		if cmp.HashSet.NsPerOp <= 0 {
			t.Errorf("%s: hashset ns/op must be positive", cmp.Scenario)
		}
		if cmp.Bloom.BytesPerItem <= 0 {
			t.Errorf("%s: bloom bytes/item must be positive", cmp.Scenario)
		}
		if cmp.HashSet.BytesPerItem <= 0 {
			t.Errorf("%s: hashset bytes/item must be positive", cmp.Scenario)
		}
	}
}

func TestCompareAddAllocatesLessThanHashSet(t *testing.T) {
	cfg := Config{Bloom: bloom.TargetFilter(1_000, 0.01),
		LookupRepeats:     1,
	}
	results, err := Compare(cfg)
	if err != nil {
		t.Fatal(err)
	}
	var add Comparison
	for _, cmp := range results {
		if cmp.Scenario == ScenarioAdd {
			add = cmp
			break
		}
	}
	if add.HashSet.AllocsPerOp <= add.Bloom.AllocsPerOp {
		t.Fatalf("add: hash set allocs/op %.2f should exceed bloom %.2f",
			add.HashSet.AllocsPerOp, add.Bloom.AllocsPerOp)
	}
	if add.AllocRatio() <= 1 {
		t.Fatalf("add: AllocRatio = %.2f, want > 1", add.AllocRatio())
	}
}

func TestCompareHashSweep(t *testing.T) {
	cfg := Config{Bloom: bloom.TargetFilter(2_000, 0.01), LookupRepeats: 1}
	strategies := []bloom.Strategy{bloom.HashFNV, bloom.HashMurmur3, bloom.HashXXHash, bloom.HashWyhash}
	results, err := CompareHashSweep(cfg, strategies)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != len(strategies) {
		t.Fatalf("got %d sweep results, want %d", len(results), len(strategies))
	}
	for i, cmp := range results {
		if cmp.Scenario != ScenarioAdd {
			t.Fatalf("sweep[%d]: scenario = %q, want add", i, cmp.Scenario)
		}
		if cmp.Bloom.NsPerOp <= 0 {
			t.Fatalf("sweep[%d]: bloom ns/op must be positive", i)
		}
	}
}

func TestCompareHashSweepInvalidStrategy(t *testing.T) {
	cfg := Config{Bloom: bloom.TargetFilter(100, 0.01)}
	_, err := CompareHashSweep(cfg, nil)
	if err == nil {
		t.Fatal("expected error for empty strategies")
	}
}

func TestParseHashStrategies(t *testing.T) {
	strategies, err := ParseHashStrategies("fnv, murmur3 ,xxhash,wyhash")
	if err != nil {
		t.Fatal(err)
	}
	if len(strategies) != 4 {
		t.Fatalf("got %d strategies, want 4", len(strategies))
	}
	if strategies[3] != bloom.HashWyhash {
		t.Fatalf("strategies[3] = %v, want wyhash", strategies[3])
	}
	_, err = ParseHashStrategies("")
	if err == nil {
		t.Fatal("expected error for empty string")
	}
}

func TestFormatHashSweep(t *testing.T) {
	cfg := Config{Bloom: bloom.TargetFilter(500, 0.01), LookupRepeats: 1}
	strategies := []bloom.Strategy{bloom.HashFNV, bloom.HashXXHash}
	results, err := CompareHashSweep(cfg, strategies)
	if err != nil {
		t.Fatal(err)
	}
	text := FormatHashSweep(cfg, strategies, results)
	if !strings.Contains(text, "hash sweep") {
		t.Fatal("sweep report missing title")
	}
	if !strings.Contains(text, "xxhash") {
		t.Fatal("sweep report missing xxhash row")
	}
	md := FormatHashSweepMarkdown(cfg, strategies, results)
	if !strings.Contains(md, "| Hash |") {
		t.Fatal("sweep markdown missing header")
	}
}

func TestCompareWithMurmur3Hash(t *testing.T) {
	cfg := Config{
		Bloom: bloom.TargetFilter(500, 0.01,
			bloom.WithFilterHash(bloom.HashMurmur3),
			bloom.WithFilterSeed(7),
		),
		LookupRepeats: 1,
	}
	results, err := Compare(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != len(AllScenarios) {
		t.Fatalf("got %d results, want %d", len(results), len(AllScenarios))
	}
}

func TestCompareFPRSweep(t *testing.T) {
	cfg := Config{Bloom: bloom.TargetFilter(2_000, 0.01), LookupRepeats: 1}
	rates := []float64{0.01, 0.05, 0.1}
	results, err := CompareFPRSweep(cfg, rates)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != len(rates) {
		t.Fatalf("got %d sweep results, want %d", len(results), len(rates))
	}
	prevBloomBytes := 0.0
	for i, cmp := range results {
		if cmp.Scenario != ScenarioAdd {
			t.Fatalf("sweep[%d]: scenario = %q, want add", i, cmp.Scenario)
		}
		if i > 0 && cmp.Bloom.BytesPerItem >= prevBloomBytes {
			t.Fatalf("sweep[%d]: bloom bytes/item %.1f should shrink as p rises (prev %.1f)",
				i, cmp.Bloom.BytesPerItem, prevBloomBytes)
		}
		prevBloomBytes = cmp.Bloom.BytesPerItem
		if cmp.HashSet.BytesPerItem <= 0 {
			t.Fatalf("sweep[%d]: hash set bytes/item must be positive", i)
		}
	}
}

func TestCompareSizeSweep(t *testing.T) {
	cfg := Config{Bloom: bloom.TargetFilter(100_000, 0.01), LookupRepeats: 1}
	counts := []uint64{500, 2_000, 5_000}
	results, err := CompareSizeSweep(cfg, counts)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != len(counts) {
		t.Fatalf("got %d sweep results, want %d", len(results), len(counts))
	}
	firstBloomBytes := 0.0
	for i, cmp := range results {
		if cmp.Scenario != ScenarioAdd {
			t.Fatalf("sweep[%d]: scenario = %q, want add", i, cmp.Scenario)
		}
		if cmp.Bloom.BytesPerItem <= 0 {
			t.Fatalf("sweep[%d]: bloom bytes/item must be positive", i)
		}
		if cmp.HashSet.BytesPerItem <= 0 {
			t.Fatalf("sweep[%d]: hash set bytes/item must be positive", i)
		}
		if cmp.SpaceRatio() <= 1 {
			t.Fatalf("sweep[%d]: space ratio %.2f, want > 1 (bloom smaller than hash set)", i, cmp.SpaceRatio())
		}
		if i == 0 {
			firstBloomBytes = cmp.Bloom.BytesPerItem
			continue
		}
		delta := cmp.Bloom.BytesPerItem - firstBloomBytes
		if delta < 0 {
			delta = -delta
		}
		if delta/firstBloomBytes > 0.05 {
			t.Fatalf("sweep[%d]: bloom bytes/item %.1f drifted from first %.1f at fixed p",
				i, cmp.Bloom.BytesPerItem, firstBloomBytes)
		}
	}
}

func TestCompareContainsMixedHitRatio(t *testing.T) {
	cfg := Config{Bloom: bloom.TargetFilter(200, 0.01),
		LookupRepeats:     1,
		LookupHitRatio:    0.5,
	}
	results, err := Compare(cfg)
	if err != nil {
		t.Fatal(err)
	}
	var mixed Comparison
	for _, cmp := range results {
		if cmp.Scenario == ScenarioContainsMixed {
			mixed = cmp
			break
		}
	}
	if mixed.LookupHitRatio != 0.5 {
		t.Fatalf("LookupHitRatio = %v, want 0.5", mixed.LookupHitRatio)
	}
	if mixed.Bloom.NsPerOp <= 0 || mixed.HashSet.NsPerOp <= 0 {
		t.Fatal("contains_mixed: ns/op must be positive")
	}
}

func TestCompareLookupMixSweep(t *testing.T) {
	cfg := Config{Bloom: bloom.TargetFilter(2_000, 0.01), LookupRepeats: 1}
	ratios := []float64{0, 0.5, 1.0}
	results, err := CompareLookupMixSweep(cfg, ratios)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != len(ratios) {
		t.Fatalf("got %d sweep results, want %d", len(results), len(ratios))
	}
	for i, cmp := range results {
		if cmp.Scenario != ScenarioContainsMixed {
			t.Fatalf("sweep[%d]: scenario = %q, want contains_mixed", i, cmp.Scenario)
		}
		if cmp.LookupHitRatio != ratios[i] {
			t.Fatalf("sweep[%d]: LookupHitRatio = %v, want %v", i, cmp.LookupHitRatio, ratios[i])
		}
	}
}

func TestCompareLookupMixSweepInvalidRatio(t *testing.T) {
	cfg := Config{Bloom: bloom.TargetFilter(100, 0.01)}
	_, err := CompareLookupMixSweep(cfg, []float64{-0.1, 0.5})
	if err == nil {
		t.Fatal("expected error for negative ratio")
	}
	_, err = CompareLookupMixSweep(cfg, nil)
	if err == nil {
		t.Fatal("expected error for empty ratios")
	}
}

func TestParseLookupMixRatios(t *testing.T) {
	ratios, err := ParseLookupMixRatios("0, 0.5 ,1")
	if err != nil {
		t.Fatal(err)
	}
	if len(ratios) != 3 {
		t.Fatalf("got %d ratios, want 3", len(ratios))
	}
	if ratios[1] != 0.5 {
		t.Fatalf("ratios[1] = %v, want 0.5", ratios[1])
	}
	_, err = ParseLookupMixRatios("")
	if err == nil {
		t.Fatal("expected error for empty string")
	}
}

func TestFormatLookupMixSweep(t *testing.T) {
	cfg := Config{Bloom: bloom.TargetFilter(500, 0.01), LookupRepeats: 1}
	ratios := []float64{0, 1.0}
	results, err := CompareLookupMixSweep(cfg, ratios)
	if err != nil {
		t.Fatal(err)
	}
	text := FormatLookupMixSweep(cfg, ratios, results)
	if !strings.Contains(text, "lookup mix sweep") {
		t.Fatal("sweep report missing title")
	}
	if !strings.Contains(text, "100%") {
		t.Fatal("sweep report missing 100% hit ratio row")
	}
	md := FormatLookupMixSweepMarkdown(cfg, ratios, results)
	if !strings.Contains(md, "| Hit ratio |") {
		t.Fatal("sweep markdown missing header")
	}
}

func TestCompareKeyLengthSweep(t *testing.T) {
	cfg := Config{Bloom: bloom.TargetFilter(500, 0.01), LookupRepeats: 1}
	lengths := []int{16, 64, 256}
	results, err := CompareKeyLengthSweep(cfg, lengths)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != len(lengths) {
		t.Fatalf("got %d sweep results, want %d", len(results), len(lengths))
	}
	prevSpace := 0.0
	for i, cmp := range results {
		if cmp.Scenario != ScenarioAdd {
			t.Fatalf("sweep[%d]: scenario = %q, want add", i, cmp.Scenario)
		}
		if cmp.KeyLength != lengths[i] {
			t.Fatalf("sweep[%d]: KeyLength = %d, want %d", i, cmp.KeyLength, lengths[i])
		}
		if cmp.Bloom.BytesPerItem <= 0 {
			t.Fatalf("sweep[%d]: bloom bytes/item must be positive", i)
		}
		if cmp.HashSet.BytesPerItem <= 0 {
			t.Fatalf("sweep[%d]: hash set bytes/item must be positive", i)
		}
		if i > 0 && cmp.SpaceRatio() <= prevSpace {
			t.Fatalf("sweep[%d]: space ratio %.2f should rise with key length (prev %.2f)",
				i, cmp.SpaceRatio(), prevSpace)
		}
		prevSpace = cmp.SpaceRatio()
		firstBloom := results[0].Bloom.BytesPerItem
		delta := cmp.Bloom.BytesPerItem - firstBloom
		if delta < 0 {
			delta = -delta
		}
		if delta/firstBloom > 0.05 {
			t.Fatalf("sweep[%d]: bloom bytes/item %.1f drifted from first %.1f at fixed n/p",
				i, cmp.Bloom.BytesPerItem, firstBloom)
		}
	}
}

func TestCompareKeyLengthSweepInvalidLength(t *testing.T) {
	cfg := Config{Bloom: bloom.TargetFilter(100, 0.01)}
	_, err := CompareKeyLengthSweep(cfg, []int{0, 16})
	if err == nil {
		t.Fatal("expected error for zero key length")
	}
	_, err = CompareKeyLengthSweep(cfg, []int{4})
	if err == nil {
		t.Fatal("expected error for key length too short for n")
	}
	_, err = CompareKeyLengthSweep(cfg, nil)
	if err == nil {
		t.Fatal("expected error for empty lengths")
	}
}

func TestParseKeyLengths(t *testing.T) {
	lengths, err := ParseKeyLengths("16, 64 ,256")
	if err != nil {
		t.Fatal(err)
	}
	if len(lengths) != 3 {
		t.Fatalf("got %d lengths, want 3", len(lengths))
	}
	if lengths[1] != 64 {
		t.Fatalf("lengths[1] = %d, want 64", lengths[1])
	}
	_, err = ParseKeyLengths("")
	if err == nil {
		t.Fatal("expected error for empty string")
	}
}

func TestFormatKeyLengthSweep(t *testing.T) {
	cfg := Config{Bloom: bloom.TargetFilter(500, 0.01), LookupRepeats: 1}
	lengths := []int{16, 256}
	results, err := CompareKeyLengthSweep(cfg, lengths)
	if err != nil {
		t.Fatal(err)
	}
	text := FormatKeyLengthSweep(cfg, lengths, results)
	if !strings.Contains(text, "key length sweep") {
		t.Fatal("sweep report missing title")
	}
	if !strings.Contains(text, "256 B") {
		t.Fatal("sweep report missing second length")
	}
	md := FormatKeyLengthSweepMarkdown(cfg, lengths, results)
	if !strings.Contains(md, "| Key length |") {
		t.Fatal("sweep markdown missing header")
	}
}

func TestMakeKeysWithLength(t *testing.T) {
	keys := makeKeysWithLength(10, 32)
	if len(keys) != 10 {
		t.Fatalf("got %d keys, want 10", len(keys))
	}
	for i, key := range keys {
		if len(key) != 32 {
			t.Fatalf("keys[%d] len = %d, want 32", i, len(key))
		}
	}
	seen := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		if _, ok := seen[string(key)]; ok {
			t.Fatal("duplicate key generated")
		}
		seen[string(key)] = struct{}{}
	}
}

func TestMakeMixedLookupKeys(t *testing.T) {
	keys := makeMixedLookupKeys(100, 0.5)
	if len(keys) != 100 {
		t.Fatalf("got %d keys, want 100", len(keys))
	}
	hits, misses := 0, 0
	for _, key := range keys {
		if strings.HasPrefix(string(key), "key-") {
			hits++
		} else if strings.HasPrefix(string(key), "miss-") {
			misses++
		} else {
			t.Fatalf("unexpected key %q", key)
		}
	}
	if hits != 50 || misses != 50 {
		t.Fatalf("hits=%d misses=%d, want 50 each", hits, misses)
	}
}

func TestCompareSizeSweepInvalidCount(t *testing.T) {
	cfg := Config{Bloom: bloom.TargetFilter(100_000, 0.01)}
	_, err := CompareSizeSweep(cfg, []uint64{0, 100})
	if err == nil {
		t.Fatal("expected error for zero item count")
	}
	_, err = CompareSizeSweep(cfg, nil)
	if err == nil {
		t.Fatal("expected error for empty counts")
	}
}

func TestParseSizeCounts(t *testing.T) {
	counts, err := ParseSizeCounts("1000, 5000 ,10000")
	if err != nil {
		t.Fatal(err)
	}
	if len(counts) != 3 {
		t.Fatalf("got %d counts, want 3", len(counts))
	}
	if counts[1] != 5_000 {
		t.Fatalf("counts[1] = %d, want 5000", counts[1])
	}
	_, err = ParseSizeCounts("")
	if err == nil {
		t.Fatal("expected error for empty string")
	}
}

func TestFormatSizeSweep(t *testing.T) {
	cfg := Config{Bloom: bloom.TargetFilter(100_000, 0.01), LookupRepeats: 1}
	counts := []uint64{500, 2_000}
	results, err := CompareSizeSweep(cfg, counts)
	if err != nil {
		t.Fatal(err)
	}
	text := FormatSizeSweep(cfg, counts, results)
	if !strings.Contains(text, "size sweep") {
		t.Fatal("sweep report missing title")
	}
	if !strings.Contains(text, "2000") {
		t.Fatal("sweep report missing second count")
	}
	md := FormatSizeSweepMarkdown(cfg, counts, results)
	if !strings.Contains(md, "| Items n |") {
		t.Fatal("sweep markdown missing header")
	}
}

func TestCompareFPRSweepInvalidRate(t *testing.T) {
	cfg := Config{Bloom: bloom.TargetFilter(100, 0.01)}
	_, err := CompareFPRSweep(cfg, []float64{0, 0.01})
	if err == nil {
		t.Fatal("expected error for zero FPR rate")
	}
	_, err = CompareFPRSweep(cfg, nil)
	if err == nil {
		t.Fatal("expected error for empty rates")
	}
}

func TestParseFPRRates(t *testing.T) {
	rates, err := ParseFPRRates("0.001, 0.01 ,0.1")
	if err != nil {
		t.Fatal(err)
	}
	if len(rates) != 3 {
		t.Fatalf("got %d rates, want 3", len(rates))
	}
	if rates[1] != 0.01 {
		t.Fatalf("rates[1] = %v, want 0.01", rates[1])
	}
	_, err = ParseFPRRates("")
	if err == nil {
		t.Fatal("expected error for empty string")
	}
}

func TestFormatFPRSweep(t *testing.T) {
	cfg := Config{Bloom: bloom.TargetFilter(500, 0.01), LookupRepeats: 1}
	rates := []float64{0.01, 0.1}
	results, err := CompareFPRSweep(cfg, rates)
	if err != nil {
		t.Fatal(err)
	}
	text := FormatFPRSweep(cfg, rates, results)
	if !strings.Contains(text, "FPR sweep") {
		t.Fatal("sweep report missing title")
	}
	if !strings.Contains(text, "0.0100") {
		t.Fatal("sweep report missing first rate")
	}
	md := FormatFPRSweepMarkdown(cfg, rates, results)
	if !strings.Contains(md, "| Target p |") {
		t.Fatal("sweep markdown missing header")
	}
}

func TestCompareRemoveUsesCountingFilter(t *testing.T) {
	cfg := Config{Bloom: bloom.TargetFilter(500, 0.01),
		LookupRepeats:     1,
	}
	results, err := Compare(cfg)
	if err != nil {
		t.Fatal(err)
	}
	var remove Comparison
	for _, cmp := range results {
		if cmp.Scenario == ScenarioRemove {
			remove = cmp
			break
		}
	}
	if remove.Bloom.NsPerOp <= 0 {
		t.Fatal("remove: bloom ns/op must be positive")
	}
	if remove.HashSet.NsPerOp <= 0 {
		t.Fatal("remove: hashset ns/op must be positive")
	}
	if remove.Bloom.BytesPerItem <= 0 {
		t.Fatal("remove: bloom bytes/item must be positive")
	}
}

func TestCompareMixedStreamDetectsDuplicates(t *testing.T) {
	cfg := Config{Bloom: bloom.TargetFilter(100, 0.01),
		LookupRepeats:     1,
	}
	results, err := Compare(cfg)
	if err != nil {
		t.Fatal(err)
	}
	var mixed Comparison
	for _, cmp := range results {
		if cmp.Scenario == ScenarioMixedStream {
			mixed = cmp
			break
		}
	}
	if mixed.HashSet.FalsePositives != 50 {
		t.Fatalf("hash set dup calls = %d, want 50", mixed.HashSet.FalsePositives)
	}
	if mixed.HashSet.FalsePositives != mixed.Bloom.FalsePositives {
		t.Fatalf("bloom dup calls = %d, hashset = %d; stream shape should match",
			mixed.Bloom.FalsePositives, mixed.HashSet.FalsePositives)
	}
}

func TestCompareInvalidConfig(t *testing.T) {
	_, err := Compare(Config{Bloom: bloom.TargetFilter(0, 0.01)})
	if err == nil {
		t.Fatal("expected error for zero capacity")
	}
	_, err = Compare(Config{Bloom: bloom.TargetFilter(100, 0)})
	if err == nil {
		t.Fatal("expected error for invalid FPR")
	}
}

func TestSpeedAndSpaceRatios(t *testing.T) {
	cmp := Comparison{
		Scenario: ScenarioAdd,
		Bloom:    BackendResult{NsPerOp: 100, BytesPerItem: 10, AllocsPerOp: 0.1},
		HashSet:  BackendResult{NsPerOp: 200, BytesPerItem: 50, AllocsPerOp: 2.0},
	}
	if ratio := cmp.SpeedRatio(); ratio != 2 {
		t.Fatalf("SpeedRatio = %v, want 2", ratio)
	}
	if ratio := cmp.SpaceRatio(); ratio != 5 {
		t.Fatalf("SpaceRatio = %v, want 5", ratio)
	}
	if ratio := cmp.AllocRatio(); ratio != 20 {
		t.Fatalf("AllocRatio = %v, want 20", ratio)
	}
}

func TestFormatReportContainsScenarios(t *testing.T) {
	cfg := Config{Bloom: bloom.TargetFilter(500, 0.01), LookupRepeats: 1}
	results, err := Compare(cfg)
	if err != nil {
		t.Fatal(err)
	}
	text := FormatReport(cfg, results)
	for _, sc := range AllScenarios {
		if !strings.Contains(text, string(sc)) {
			t.Errorf("report missing scenario %q", sc)
		}
	}
	if !strings.Contains(text, "Speedup > 1") {
		t.Error("report missing legend")
	}
	if !strings.Contains(text, "allocs/op") {
		t.Error("report missing allocation columns")
	}
}

func TestFormatMarkdownTable(t *testing.T) {
	cfg := Config{Bloom: bloom.TargetFilter(500, 0.01), LookupRepeats: 1}
	results, err := Compare(cfg)
	if err != nil {
		t.Fatal(err)
	}
	md := FormatMarkdown(cfg, results)
	if !strings.Contains(md, "| Scenario |") {
		t.Fatal("markdown missing header row")
	}
	if !strings.Contains(md, string(ScenarioMixedStream)) {
		t.Fatal("markdown missing mixed_stream row")
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Bloom.ExpectedCapacity() != 100_000 {
		t.Fatalf("ExpectedCapacity = %d, want 100000", cfg.Bloom.ExpectedCapacity())
	}
	if cfg.Bloom.FalsePositiveRate() != 0.01 {
		t.Fatalf("FalsePositiveRate = %v, want 0.01", cfg.Bloom.FalsePositiveRate())
	}
	if cfg.LookupHitRatio != 0.5 {
		t.Fatalf("LookupHitRatio = %v, want 0.5", cfg.LookupHitRatio)
	}
}
