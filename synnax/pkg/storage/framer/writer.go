package framer

import (
	"context"
	"github.com/cockroachdb/errors"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/synnax/pkg/storage/control"
	"github.com/synnaxlabs/x/telem"
)

type WriterConfig struct {
	Start       telem.TimeStamp
	Channels    []ChannelKey
	Authorities []control.Authority
}

func (c WriterConfig) cesium() cesium.WriterConfig {
	return cesium.WriterConfig{Start: c.Start, Channels: c.Channels}
}

type Writer struct {
	internal *cesium.Writer
	gate     *control.Gate[ChannelKey]
	err      error
	relay    *relay
}

func (db *DB) OpenWriter(ctx context.Context, cfg WriterConfig) (*Writer, error) {
	internal, err := db.internal.OpenWriter(ctx, cfg.cesium())
	if err != nil {
		return nil, err
	}
	g := db.control.OpenGate(cfg.Start.Range(telem.TimeStampMax))
	g.Set(cfg.Channels, cfg.Authorities)
	return &Writer{internal: internal, gate: g, relay: db.relay}, nil
}

func (w *Writer) Write(ctx context.Context, fr Frame) bool {
	failed := w.gate.Check(fr.Keys)
	if len(failed) > 0 {
		w.err = errors.New("write failed - insufficient permissions")
		return false
	}
	alignedFr, ok := w.internal.Write(ctx, fr)
	if !ok {
		return false
	}
	w.relay.inlet.Inlet() <- alignedFr
	return true
}

func (w *Writer) SetAuthorities(ctx context.Context, keys []ChannelKey, auth []control.Authority) {
	w.gate.Set(keys, auth)
}

func (w *Writer) Close() error {
	return errors.CombineErrors(w.err, w.internal.Close())
}

func (w *Writer) Error() error {
	return errors.CombineErrors(w.err, w.internal.Error())
}

func (w *Writer) Commit(ctx context.Context) (telem.TimeStamp, bool) {
	return w.internal.Commit(ctx)
}
