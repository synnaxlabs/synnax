package cesium

import (
	"context"
	"github.com/cockroachdb/errors"
	"github.com/samber/lo"
	"github.com/synnaxlabs/cesium/internal/controller"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
)

type ControlUpdate struct {
	Transfers []controller.Transfer `json:"transfers"`
}

func (db *DB) ConfigureControlDigestChannel(ctx context.Context, key ChannelKey) error {
	ch, err := db.RetrieveChannel(ctx, key)
	if errors.Is(err, ChannelNotFound) {
		ch.Key = key
		ch.DataType = telem.StringT
		ch.Virtual = true
		if err := db.CreateChannel(ctx, ch); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}
	db.digests.key = key
	w, err := db.NewStreamWriter(ctx, WriterConfig{
		Start:    telem.Now(),
		Channels: []ChannelKey{key},
	})
	if err != nil {
		return err
	}
	db.digests.inlet, db.digests.outlet = confluence.Attach[WriterRequest, WriterResponse](w, 100)
	sCtx, _ := signal.Isolated()
	w.Flow(sCtx, confluence.CloseInletsOnExit())
	return nil
}

func (db *DB) updateControlDigests(
	ctx context.Context,
	u ControlUpdate,
) {
	if db.digests.key == 0 {
		return
	}
	db.digests.inlet.Inlet() <- WriterRequest{
		Command: WriterWrite,
		Frame: Frame{
			Keys: []ChannelKey{db.digests.key},
			Series: []telem.Series{
				telem.Series{
					DataType: telem.StringT,
					Data:     lo.Must((&binary.JSONEncoderDecoder{}).Encode(ctx, u)),
				},
			},
		},
	}
}

func (db *DB) closeControlDigests() {
	if db.digests.key != 0 {
		db.digests.inlet.Close()
		confluence.Drain(db.digests.outlet)
	}
}
