package bloom

import "time"

// MeasureDeriveNsPerOp estimates nanoseconds per Hasher.Derive call for the given
// sample keys. It is intended for relative comparisons between strategies on the
// same machine, not as an absolute benchmark.
func MeasureDeriveNsPerOp(h Hasher, samples int, keyFor func(i int) []byte) float64 {
	if samples <= 0 {
		samples = 256
	}
	if keyFor == nil {
		return 0
	}

	// Warm up caches and branch predictors.
	for i := 0; i < samples; i++ {
		h.Derive(keyFor(i))
	}

	const minDuration = 5 * time.Millisecond
	start := time.Now()
	iterations := 0
	for time.Since(start) < minDuration {
		for i := 0; i < samples; i++ {
			h.Derive(keyFor(i))
			iterations++
		}
	}
	if iterations == 0 {
		return 0
	}
	return float64(time.Since(start).Nanoseconds()) / float64(iterations)
}
