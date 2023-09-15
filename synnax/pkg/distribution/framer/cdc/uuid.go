package cdc

import (
	"context"
	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	"io"
)

type UUIDConfig[E gorp.Entry[uuid.UUID]] struct {
	Set    channel.Channel
	Delete channel.Channel
	Framer framer.Writable
	DB     *gorp.DB
}

func OpenUUID[E gorp.Entry[uuid.UUID]](ctx context.Context, cfg UUIDConfig[E]) (io.Closer, error) {
	channels := []channel.Channel{cfg.Set, cfg.Delete}
	keys := channel.KeysFromChannels(channels)
	w, err := cfg.Framer.NewStreamWriter(ctx, framer.WriterConfig{
		Keys:  keys,
		Start: telem.Now(),
	})
	if err != nil {
		return nil, err
	}
	sCtx, cancel := signal.Isolated()
	defer cancel()
	p := plumber.New()
	t := &confluence.TransformSubscriber[gorp.TxReader[uuid.UUID, E], framer.WriterRequest]{
		Observable: gorp.Observe[uuid.UUID, E](cfg.DB),
		Transform: func(ctx context.Context, r gorp.TxReader[uuid.UUID, E]) (framer.WriterRequest, bool, error) {
			var ids []byte
			for c, ok, _ := r.Next(ctx); ok; c, ok, _ = r.Next(ctx) {
				b, _ := c.Value.GorpKey().MarshalBinary()
				ids = append(ids, b...)
			}
			f := framer.Frame{
				Keys: channel.Keys{channels[0].Key()},
				Series: []telem.Series{{
					DataType: telem.UUIDT,
					Data:     ids,
				}},
			}
			return framer.WriterRequest{Command: writer.Data, Frame: f}, true, nil
		},
	}
	plumber.SetSource[framer.WriterRequest](p, "source", t)
	plumber.SetSink[framer.WriterRequest](p, "sink", w)
	p.Flow(sCtx)
	return signal.NewShutdown(sCtx, cancel), nil
}
