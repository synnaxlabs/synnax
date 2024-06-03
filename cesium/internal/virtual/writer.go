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
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

var WriterClosedError = core.EntityClosed("virtual.writer")

type WriterConfig struct {
	Subject           control.Subject
	Start             telem.TimeStamp
	End               telem.TimeStamp
	Authority         control.Authority
	ErrOnUnauthorized *bool
}

var (
	_                   config.Config[WriterConfig] = WriterConfig{}
	DefaultWriterConfig                             = WriterConfig{}
)

func (cfg WriterConfig) Validate() error {
	v := validate.New("virtual.WriterConfig")
	validate.NotEmptyString(v, "Subject.Key", cfg.Subject.Key)
	validate.NotNil(v, "ErrONUnauthorized", cfg.ErrOnUnauthorized)
	return v.Error()
}

func (cfg WriterConfig) Override(other WriterConfig) WriterConfig {
	cfg.Start = override.Zero(cfg.Start, other.Start)
	cfg.End = override.Zero(cfg.End, other.End)
	cfg.Subject = override.If(cfg.Subject, other.Subject, other.Subject.Key != "")
	cfg.Authority = override.Numeric(cfg.Authority, other.Authority)
	cfg.ErrOnUnauthorized = override.Nil(cfg.ErrOnUnauthorized, other.ErrOnUnauthorized)
	return cfg
}

func (cfg WriterConfig) domain() telem.TimeRange {
	return telem.TimeRange{Start: cfg.Start, End: lo.Ternary(cfg.End.IsZero(), telem.TimeStampMax, cfg.End)}
}

type Writer struct {
	Channel core.Channel
	onClose func()
	control *controller.Gate[*controlEntity]
	closed  bool
	WriterConfig
}

func (db *DB) OpenWriter(_ context.Context, cfgs ...WriterConfig) (w *Writer, transfer controller.Transfer, err error) {
	cfg, err := config.New(DefaultWriterConfig, cfgs...)
	if err != nil {
		return nil, transfer, err
	}
	w = &Writer{WriterConfig: cfg, Channel: db.Channel, onClose: func() { db.mu.Add(-1) }}
	gateCfg := controller.GateConfig{
		TimeRange: cfg.domain(),
		Authority: cfg.Authority,
		Subject:   cfg.Subject,
	}
	var g *controller.Gate[*controlEntity]
	g, transfer, err = db.controller.OpenGateAndMaybeRegister(gateCfg, func() (*controlEntity, error) {
		a := telem.AlignmentPair(0)
		return &controlEntity{
			ck:    db.Channel.Key,
			align: a,
		}, nil
	})
	if err != nil {
		return nil, transfer, err
	}
	if *cfg.ErrOnUnauthorized {
		if _, err = g.Authorize(); err != nil {
			g.Release()
			return nil, transfer, err
		}
	}
	w.control = g
	db.mu.Add(1)
	return w, transfer, err
}

func (w *Writer) Write(series telem.Series) (telem.AlignmentPair, error) {
	if w.closed {
		return 0, WriterClosedError
	}
	if err := w.Channel.ValidateSeries(series); err != nil {
		return 0, err
	}
	e, ok := w.control.Authorized()
	if !ok {
		return 0, nil
	}
	a := e.align
	if series.DataType.Density() != telem.DensityUnknown {
		e.align += telem.AlignmentPair(series.Len())
	}
	return a, nil
}

func (w *Writer) SetAuthority(a control.Authority) controller.Transfer {
	return w.control.SetAuthority(a)
}

func (w *Writer) Close() (controller.Transfer, error) {
	if w.closed {
		return controller.Transfer{}, nil
	}
	w.closed = true
	_, t := w.control.Release()
	w.onClose()
	return t, nil
}
