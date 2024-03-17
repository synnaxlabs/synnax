package unary

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/synnaxlabs/cesium/internal/controller"
	"github.com/synnaxlabs/cesium/internal/domain"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/telem"
)

func (db *DB) Delete(ctx context.Context, tr telem.TimeRange) error {
	gateCfg := controller.GateConfig{
		TimeRange: tr,
		Authority: 0,
		Subject:   control.Subject{Key: uuid.New().String(), Name: "Delete Writer"},
		Stealth:   true,
	}

	g, _, _, err := db.Controller.OpenGateAndMaybeRegister(gateCfg, func() (controlledWriter, error) {
		return controlledWriter{
			Writer:     nil,
			channelKey: db.Channel.Key,
		}, nil
	})

	if err != nil {
		return err
	}

	_, ok := g.Authorize()
	if !ok {
		g.Release()
		return controller.Unauthorized(g.Subject.Name, db.Channel.Key)
	}

	i := db.Domain.NewLockedIterator(domain.IterRange(db.Domain.GetBounds()))
	if ok := i.SeekGE(ctx, tr.Start); !ok {
		return errors.New("Start TS not found")
	}
	approxDist, err := db.index().Distance(ctx, telem.TimeRange{
		Start: i.TimeRange().Start,
		End:   tr.Start,
	}, false, false)
	if err != nil {
		return err
	}
	startOffset := approxDist.Upper
	startPosition := i.Position()

	if ok := i.SeekLE(ctx, tr.End); !ok {
		return errors.New("End TS not found")
	}
	approxDist, err = db.index().Distance(ctx, telem.TimeRange{
		Start: tr.End,
		End:   i.TimeRange().End,
	}, false, false)
	if err != nil {
		return err
	}
	endOffset := approxDist.Lower + 1
	endPosition := i.Position()

	err = db.Domain.Delete(ctx, startPosition, endPosition, startOffset*int64(db.Channel.DataType.Density()), endOffset*int64(db.Channel.DataType.Density()), tr)
	if err != nil {
		return err
	}

	g.Release()
	return i.Close()
}

func (db *DB) GarbageCollect(ctx context.Context, maxSizeRead uint32) error {
	if db.openIteratorWriters.Value() > 0 {
		return nil
	}
	err := db.Domain.CollectTombstone(ctx, maxSizeRead)
	return err
}
