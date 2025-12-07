package benchcompare

import "runtime"

// allocsPerOp runs fn once and returns heap allocations divided by ops.
func allocsPerOp(ops int, fn func()) float64 {
	if ops <= 0 {
		return 0
	}
	runtime.GC()
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	fn()
	runtime.ReadMemStats(&after)
	return float64(after.Mallocs-before.Mallocs) / float64(ops)
}
