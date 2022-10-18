package kv

import (
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/x/gorp"
)

// Writer is a key-value backed implementation of thbe core.MDWriter interface.
type Writer struct {
	*gorp.KVWriter[[]byte, core.SegmentMD]
}

// Write implements the core.MDWriter interface.
func (w *Writer) Write(entries []core.SegmentMD) error { return w.WriteMany(entries) }

// Commit implements the core.MDWriter interface.
func (w *Writer) Commit() error { return w.KVWriter.Commit() }
