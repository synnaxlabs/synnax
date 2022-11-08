package kv

import (
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/position"
	"github.com/synnaxlabs/x/gorp"
	"go.uber.org/zap"
)

// Batch is a key-value backed implementation of the core.MDBatch interface.
type Batch struct {
	logger *zap.Logger
	*gorp.KVBatch[[]byte, core.SegmentMD]
}

// Write implements the core.MDBatch interface.
func (w *Batch) Write(entries []core.SegmentMD) error { return w.WriteMany(entries) }

// Commit implements the core.MDBatch interface.
func (w *Batch) Commit() error { return w.KVBatch.Commit() }

func (w *Batch) NewIterator(ch core.Channel) core.PositionIterator {
	return newPositionIterator(w.KVBatch, ch, w.logger)
}

func (w *Batch) Retrieve(key core.ChannelKey, alignment position.Position) (md core.SegmentMD, err error) {
	buf := make([]byte, 11)
	core.WriteSegmentKey(key, alignment, buf)
	return md, gorp.NewRetrieve[[]byte, core.SegmentMD]().WhereKeys(buf).Entry(&md).Exec(w.KVBatch)
}
