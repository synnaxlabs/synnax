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

var errWriterClosed = core.EntityClosed("virtual.writer")

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

func (cfg WriterConfig) gateConfig() controller.GateConfig {
	return controller.GateConfig{
		TimeRange: cfg.domain(),
		Authority: cfg.Authority,
		Subject:   cfg.Subject,
	}
}

type Writer struct {
	Channel   core.Channel
	onClose   func()
	control   *controller.Gate[*controlEntity]
	wrapError func(error) error
	closed    bool
	WriterConfig
}

func (db *DB) OpenWriter(_ context.Context, cfgs ...WriterConfig) (w *Writer, transfer controller.Transfer, err error) {
	if db.closed.Load() {
		err = dbClosed
		return nil, transfer, db.wrapError(err)
	}
	cfg, err := config.New(DefaultWriterConfig, cfgs...)
	if err != nil {
		return nil, transfer, db.wrapError(err)
	}
	w = &Writer{
		WriterConfig: cfg,
		Channel:      db.Channel,
		wrapError:    db.wrapError,
		onClose:      func() { db.entityCount.decrement() },
	}
	var g *controller.Gate[*controlEntity]
	g, transfer, err = db.controller.OpenGateAndMaybeRegister(
		cfg.gateConfig(),
		func() (*controlEntity, error) {
			return &controlEntity{ck: db.Channel.Key, alignment: 0}, nil
		},
	)
	if err != nil {
		return nil, transfer, db.wrapError(err)
	}
	if *cfg.ErrOnUnauthorized {
		if _, err = g.Authorize(); err != nil {
			g.Release()
			return nil, transfer, db.wrapError(err)
		}
	}
	w.control = g
	db.entityCount.increment()
	return w, transfer, db.wrapError(err)
}

func (w *Writer) Write(series telem.Series) (telem.AlignmentPair, error) {
	if w.closed {
		return 0, w.wrapError(errWriterClosed)
	}
	if err := w.Channel.ValidateSeries(series); err != nil {
		return 0, w.wrapError(err)
	}
	e, err := w.control.Authorize()
	if err != nil {
		return 0, w.wrapError(err)
	}
	a := e.alignment
	if series.DataType.Density() != telem.DensityUnknown {
		e.alignment += telem.AlignmentPair(series.Len())
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
