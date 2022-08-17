package allocate

import "github.com/arya-analytics/x/alamos"

type Metrics struct {
	// Allocate counts the number of keys allocated by the Allocator and the average time to allocate a key.
	Allocate alamos.Duration
}

func newMetrics(exp alamos.Experiment) Metrics {
	subExp := alamos.Sub(exp, "allocate.Allocator")
	return Metrics{
		Allocate: alamos.NewGaugeDuration(subExp, alamos.Debug, "allocate"),
	}
}
