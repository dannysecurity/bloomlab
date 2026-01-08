package bloom

import "testing"

func BenchmarkHasherDeriveFNV(b *testing.B) {
	h := NewHasher(HashFNV, 0)
	key := []byte("benchmark-key-for-hash-derive")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key[0] = byte(i)
		_, _ = h.Derive(key)
	}
}

func BenchmarkHasherDeriveMurmur3(b *testing.B) {
	h := NewHasher(HashMurmur3, 0)
	key := []byte("benchmark-key-for-hash-derive")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key[0] = byte(i)
		_, _ = h.Derive(key)
	}
}

func BenchmarkHasherDeriveXXHash(b *testing.B) {
	h := NewHasher(HashXXHash, 0)
	key := []byte("benchmark-key-for-hash-derive")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key[0] = byte(i)
		_, _ = h.Derive(key)
	}
}

func BenchmarkHasherDeriveHighway(b *testing.B) {
	h := NewHasher(HashHighway, 0)
	key := []byte("benchmark-key-for-hash-derive")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key[0] = byte(i)
		_, _ = h.Derive(key)
	}
}

func BenchmarkHasherDeriveWyhash(b *testing.B) {
	h := NewHasher(HashWyhash, 0)
	key := []byte("benchmark-key-for-hash-derive")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key[0] = byte(i)
		_, _ = h.Derive(key)
	}
}

func BenchmarkFilterAddMurmur3(b *testing.B) {
	cfg := TargetConfig(100_000, 0.01, WithHash(HashMurmur3))
	f, _ := NewFilter(cfg)
	key := []byte("benchmark-key")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key[0] = byte(i)
		f.Add(key)
	}
}
