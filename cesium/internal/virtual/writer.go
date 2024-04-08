// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package virtual

import (
	"context"
	"github.com/samber/lo"
	"github.com/synnaxlabs/cesium/internal/controller"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/telem"
)

func (db *DB) OpenWriter(_ context.Context, cfg WriterConfig) (w *Writer, transfer controller.Transfer, err error) {
	w = &Writer{WriterConfig: cfg, Channel: db.Channel, decrementCounter: func() { db.openWriters.Add(-1) }}
	gateCfg := controller.GateConfig{
		TimeRange: cfg.domain(),
		Authority: cfg.Authority,
		Subject:   cfg.Subject,
	}
	var g *controller.Gate[*controlEntity]
	g, transfer, err = db.controller.OpenGateAndMaybeRegister(gateCfg, func() (*controlEntity, error) {
		a := telem.Alignment(0)
		return &controlEntity{
			ck:    db.Channel.Key,
			align: a,
		}, nil
	})
	w.control = g
	db.openWriters.Add(1)
	return w, transfer, err
}

type WriterConfig struct {
	Subject   control.Subject
	Start     telem.TimeStamp
	End       telem.TimeStamp
	Authority control.Authority
}

func (cfg WriterConfig) domain() telem.TimeRange {
	return telem.TimeRange{Start: cfg.Start, End: lo.Ternary(cfg.End.IsZero(), telem.TimeStampMax, cfg.End)}
}

type Writer struct {
	Channel          core.Channel
	decrementCounter func()
	control          *controller.Gate[*controlEntity]
	closed           bool
	WriterConfig
}

func (w *Writer) Write(series telem.Series) (telem.Alignment, error) {
	if w.closed {
		return 0, EntityClosed("virtual writer")
	}
	if err := w.Channel.ValidateSeries(series); err != nil {
		return 0, err
	}
	e, ok := w.control.Authorize()
	if !ok {
		return 0, nil
	}
	a := e.align
	if series.DataType.Density() != telem.DensityUnknown {
		e.align += telem.Alignment(series.Len())
	}
	return a, nil
}

func (w *Writer) SetAuthority(a control.Authority) controller.Transfer {
	return w.control.SetAuthority(a)
}

func (w *Writer) Close() (controller.Transfer, error) {
	if w.closed {
		return controller.Transfer{}, EntityClosed("virtual writer")
	}
	w.closed = true
	_, t := w.control.Release()
	w.decrementCounter()
	return t, nil
}
