// Copyright 2026 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/cesium/internal/control"
	"github.com/synnaxlabs/cesium/internal/resource"
	"github.com/synnaxlabs/x/config"
	xcontrol "github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

var errWriterClosed = resource.NewClosedError("virtual.writer")

type WriterConfig struct {
	ErrOnUnauthorizedOpen *bool
	Subject               xcontrol.Subject
	Start                 telem.TimeStamp
	End                   telem.TimeStamp
	Authority             xcontrol.Authority
}

var (
	_                   config.Config[WriterConfig] = WriterConfig{}
	DefaultWriterConfig                             = WriterConfig{
		ErrOnUnauthorizedOpen: config.False(),
	}
)

func (cfg WriterConfig) Validate() error {
	v := validate.New("virtual.writer_config")
	validate.NotEmptyString(v, "subject.key", cfg.Subject.Key)
	validate.NotNil(v, "err_on_unauthorized_open", cfg.ErrOnUnauthorizedOpen)
	return v.Error()
}

func (cfg WriterConfig) Override(other WriterConfig) WriterConfig {
	cfg.Start = override.Zero(cfg.Start, other.Start)
	cfg.End = override.Zero(cfg.End, other.End)
	cfg.Subject = override.If(cfg.Subject, other.Subject, other.Subject.Key != "")
	cfg.Authority = override.Numeric(cfg.Authority, other.Authority)
	cfg.ErrOnUnauthorizedOpen = override.Nil(cfg.ErrOnUnauthorizedOpen, other.ErrOnUnauthorizedOpen)
	return cfg
}

func (cfg WriterConfig) domain() telem.TimeRange {
	return telem.TimeRange{Start: cfg.Start, End: lo.Ternary(cfg.End.IsZero(), telem.TimeStampMax, cfg.End)}
}

type Writer struct {
	// onClose is called when the writer is closed.
	onClose func()
	// control stores the control gate held by the virtual writer, and used to track control
	// handoff scenarios with other writers.
	control *control.Gate[*controlResource]
	// wrapError is a function that wraps any error originating from this writer to
	// provide context including the writer's channel key and name.
	wrapError func(error) error
	// Channel stores information about the channel being written to, most importantly
	// the density and index.
	Channel channel.Channel
	WriterConfig
	// closed stores whether the writer is closed. Operations like Write and Commit do
	// not succeed on closed writers.
	closed bool
}

func (db *DB) OpenWriter(_ context.Context, cfgs ...WriterConfig) (w *Writer, transfer control.Transfer, err error) {
	if db.closed.Load() {
		err = ErrDBClosed
		return nil, transfer, db.wrapError(err)
	}
	cfg, err := config.New(DefaultWriterConfig, cfgs...)
	if err != nil {
		return nil, transfer, db.wrapError(err)
	}
	w = &Writer{
		WriterConfig: cfg,
		Channel:      db.cfg.Channel,
		wrapError:    db.wrapError,
	}
	if w.control, transfer, err = db.controller.OpenGate(control.GateConfig[*controlResource]{
		TimeRange:             cfg.domain(),
		ErrOnUnauthorizedOpen: cfg.ErrOnUnauthorizedOpen,
		Authority:             cfg.Authority,
		Subject:               cfg.Subject,
		OpenResource: func() (*controlResource, error) {
			return &controlResource{
				ck:        db.cfg.Channel.Key,
				alignment: telem.NewAlignment(db.leadingAlignment.Add(1), 0),
			}, nil
		},
	}); err != nil {
		return nil, transfer, db.wrapError(err)
	}
	db.openWriters.Add(1)
	w.onClose = func() {
		db.openWriters.Add(-1)
	}
	return w, transfer, nil
}

func (w *Writer) Write(series telem.Series) (telem.Alignment, error) {
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
	// copy the alignment here because we want to return the alignment of the FIRST
	// sample, not the last.
	a := e.alignment
	e.alignment = e.alignment.AddSamples(uint32(series.Len()))
	return a, nil
}

func (w *Writer) SetAuthority(a xcontrol.Authority) control.Transfer {
	return w.control.SetAuthority(a)
}

func (w *Writer) Close() (control.Transfer, error) {
	if w.closed {
		return control.Transfer{}, nil
	}
	w.closed = true
	_, t := w.control.Release()
	w.onClose()
	return t, nil
}
