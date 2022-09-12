package kfs

import (
	"github.com/synnaxlabs/x/alamos"
)

type Metrics struct {
	// Acquire tracks the number of times a File is acquired, and the average time to Acquire a File.
	Acquire alamos.Duration
	// Release tracks the number of times a File is released, and the average time to Release a File.
	Release alamos.Duration
	// Delete tracks the number of files deleted, and the average time to delete a file.
	Delete alamos.Duration
	// Close tracks the number of files closed, and the average time to close a file.
	Close alamos.Duration
}

func newMetrics(exp alamos.Experiment) Metrics {
	subExp := alamos.Sub(exp, "kfs.fs")
	return Metrics{
		Acquire: alamos.NewGaugeDuration(subExp, alamos.Debug, "TryLock"),
		Release: alamos.NewGaugeDuration(subExp, alamos.Debug, "Unlock"),
		Delete:  alamos.NewGaugeDuration(subExp, alamos.Debug, "Remove"),
		Close:   alamos.NewGaugeDuration(subExp, alamos.Debug, "Shutdown"),
	}
}
