package benchcompare

import (
	"strings"
	"testing"
)

func TestCompareAllScenarios(t *testing.T) {
	cfg := Config{
		ItemCount:         2_000,
		FalsePositiveRate: 0.01,
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

func TestCompareMixedStreamDetectsDuplicates(t *testing.T) {
	cfg := Config{
		ItemCount:         100,
		FalsePositiveRate: 0.01,
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
	_, err := Compare(Config{ItemCount: 0, FalsePositiveRate: 0.01})
	if err == nil {
		t.Fatal("expected error for zero ItemCount")
	}
	_, err = Compare(Config{ItemCount: 100, FalsePositiveRate: 0})
	if err == nil {
		t.Fatal("expected error for invalid FPR")
	}
}

func TestSpeedAndSpaceRatios(t *testing.T) {
	cmp := Comparison{
		Scenario: ScenarioAdd,
		Bloom:    BackendResult{NsPerOp: 100, BytesPerItem: 10},
		HashSet:  BackendResult{NsPerOp: 200, BytesPerItem: 50},
	}
	if ratio := cmp.SpeedRatio(); ratio != 2 {
		t.Fatalf("SpeedRatio = %v, want 2", ratio)
	}
	if ratio := cmp.SpaceRatio(); ratio != 5 {
		t.Fatalf("SpaceRatio = %v, want 5", ratio)
	}
}

func TestFormatReportContainsScenarios(t *testing.T) {
	cfg := Config{ItemCount: 500, FalsePositiveRate: 0.01, LookupRepeats: 1}
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
}

func TestFormatMarkdownTable(t *testing.T) {
	cfg := Config{ItemCount: 500, FalsePositiveRate: 0.01, LookupRepeats: 1}
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
	if cfg.ItemCount != 100_000 {
		t.Fatalf("ItemCount = %d, want 100000", cfg.ItemCount)
	}
	if cfg.FalsePositiveRate != 0.01 {
		t.Fatalf("FalsePositiveRate = %v, want 0.01", cfg.FalsePositiveRate)
	}
}
