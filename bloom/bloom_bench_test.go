package bloom

import (
	"fmt"
	"runtime"
	"testing"
)

const benchItemCount = 100_000

// Hash-set benchmarks mirror the Filter* cases so `go test -bench=. -benchmem`
// can compare bloom filter throughput and allocation against map[string]struct{}.

func BenchmarkFilterAdd(b *testing.B) {
	f, _ := New(100_000, 0.01)
	key := []byte("benchmark-key")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key[0] = byte(i)
		f.Add(key)
	}
}

func BenchmarkFilterContainsHit(b *testing.B) {
	f, _ := New(100_000, 0.01)
	key := []byte("benchmark-key")
	f.Add(key)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = f.Contains(key)
	}
}

func BenchmarkFilterContainsMiss(b *testing.B) {
	f, _ := New(100_000, 0.01)
	for i := 0; i < 1000; i++ {
		f.Add([]byte(fmt.Sprintf("seed-%d", i)))
	}
	miss := []byte("definitely-not-inserted")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = f.Contains(miss)
	}
}

func BenchmarkCountingFilterAdd(b *testing.B) {
	cf, _ := NewCountingFromTarget(100_000, 0.01)
	key := []byte("benchmark-key")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key[0] = byte(i)
		_ = cf.Add(key)
	}
}

func BenchmarkCountingFilterRemove(b *testing.B) {
	cf, _ := NewCountingFromTarget(100_000, 0.01)
	keys := make([][]byte, b.N)
	for i := range keys {
		keys[i] = []byte(fmt.Sprintf("k-%d", i))
		_ = cf.Add(keys[i])
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cf.Remove(keys[i])
	}
}

func BenchmarkMapSetAdd(b *testing.B) {
	set := make(map[string]struct{}, 100_000)
	key := []byte("benchmark-key")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key[0] = byte(i)
		set[string(key)] = struct{}{}
	}
}

func BenchmarkMapSetContainsHit(b *testing.B) {
	set := map[string]struct{}{"benchmark-key": {}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = set["benchmark-key"]
	}
}

func BenchmarkMapSetContainsMiss(b *testing.B) {
	set := make(map[string]struct{}, 1000)
	for i := 0; i < 1000; i++ {
		set[fmt.Sprintf("seed-%d", i)] = struct{}{}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = set["definitely-not-inserted"]
	}
}

// Footprint benchmarks report storage-bytes/item alongside a steady lookup loop so
// `go test -bench=Footprint -benchmem ./bloom/` compares space use at equal capacity.
func BenchmarkFilterStorageFootprint(b *testing.B) {
	f, err := New(benchItemCount, 0.01)
	if err != nil {
		b.Fatal(err)
	}
	for i := 0; i < benchItemCount; i++ {
		f.Add([]byte(fmt.Sprintf("key-%d", i)))
	}
	storageBytes := (f.BitCount() + 7) / 8
	probe := []byte("key-4242")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = f.Contains(probe)
	}
	b.StopTimer()
	b.ReportMetric(float64(storageBytes)/benchItemCount, "storage-bytes/item")
}

func BenchmarkMapSetStorageFootprint(b *testing.B) {
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	set := make(map[string]struct{}, benchItemCount)
	for i := 0; i < benchItemCount; i++ {
		set[fmt.Sprintf("key-%d", i)] = struct{}{}
	}
	runtime.ReadMemStats(&after)
	heapBytes := after.HeapAlloc - before.HeapAlloc
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = set["key-4242"]
	}
	b.StopTimer()
	b.ReportMetric(float64(heapBytes)/benchItemCount, "storage-bytes/item")
}
