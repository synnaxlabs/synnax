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

func (db *DB) OpenWriter(ctx context.Context, cfg WriterConfig) (w *Writer, transfer controller.Transfer) {
	w = &Writer{WriterConfig: cfg, Channel: db.Channel}
	gateCfg := controller.Config{
		TimeRange: cfg.domain(),
		Authority: cfg.Authority,
		Name:      cfg.Name,
	}
	var (
		g  *controller.Gate[*controlEntity]
		ok bool
	)
	g, transfer, ok = db.controller.OpenGate(gateCfg)
	if !ok {
		gateCfg.TimeRange = cfg.domain()
		a := telem.Alignment(0)
		g, transfer = db.controller.RegisterAndOpenGate(gateCfg, &controlEntity{ck: db.Channel.Key, align: a})
	}
	w.control = g
	return w, transfer
}

type WriterConfig struct {
	Name      string
	Start     telem.TimeStamp
	End       telem.TimeStamp
	Authority control.Authority
}

func (cfg WriterConfig) domain() telem.TimeRange {
	return telem.TimeRange{Start: cfg.Start, End: lo.Ternary(cfg.End.IsZero(), telem.TimeStampMax, cfg.End)}
}

type Writer struct {
	Channel core.Channel
	control *controller.Gate[*controlEntity]
	WriterConfig
}

func (w *Writer) Write(series telem.Series) (telem.Alignment, error) {
	if err := w.Channel.ValidateSeries(series); err != nil {
		return 0, err
	}
	e, ok := w.control.Authorize()
	if !ok {
		return 0, nil
	}
	a := e.align
	e.align += telem.Alignment(series.Len())
	return a, nil
}

func (w *Writer) Close() (controller.Transfer, error) {
	_, t := w.control.Release()
	return t, nil
}
