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

// ErrWriterClosed is returned when an operation is attempted on a closed writer.
var ErrWriterClosed = resource.NewClosedError("virtual.writer")

// WriterConfig configures a virtual writer.
type WriterConfig struct {
	// ErrOnUnauthorizedOpen controls whether an error is returned when the writer is
	// unauthorized to open.
	//
	// [OPTIONAL] - Defaults to false.
	ErrOnUnauthorizedOpen *bool
	// Subject identifies the writer in the channel's control system, and is reported in
	// control transfers involving the writer.
	//
	// [REQUIRED]
	Subject xcontrol.Subject
	// Start is the start of the time range the writer claims control over.
	//
	// [OPTIONAL]
	Start telem.TimeStamp
	// End is the end of the time range the writer claims control over.
	//
	// [OPTIONAL] - Defaults to telem.TimeStampMax.
	End telem.TimeStamp
	// Authority is the control authority the writer opens with. Writes are only
	// authorized while the writer holds the highest authority over the channel.
	//
	// [OPTIONAL]
	Authority xcontrol.Authority
}

var _ config.Config[WriterConfig] = WriterConfig{}

// Validate implements config.Config.
func (cfg WriterConfig) Validate() error {
	v := validate.New("virtual.writer_config")
	validate.NotEmptyString(v, "subject.key", cfg.Subject.Key)
	validate.NotNil(v, "err_on_unauthorized_open", cfg.ErrOnUnauthorizedOpen)
	return v.Error()
}

// Override implements config.Config.
func (cfg WriterConfig) Override(other WriterConfig) WriterConfig {
	cfg.Start = override.Zero(cfg.Start, other.Start)
	cfg.End = override.Zero(cfg.End, other.End)
	cfg.Subject = override.If(cfg.Subject, other.Subject, other.Subject.Key != "")
	cfg.Authority = override.Numeric(cfg.Authority, other.Authority)
	cfg.ErrOnUnauthorizedOpen = override.Nil(
		cfg.ErrOnUnauthorizedOpen, other.ErrOnUnauthorizedOpen,
	)
	return cfg
}

func (cfg WriterConfig) domain() telem.TimeRange {
	return telem.TimeRange{
		Start: cfg.Start,
		End:   lo.Ternary(cfg.End.IsZero(), telem.TimeStampMax, cfg.End),
	}
}

// A Writer is used to write to a virtual channel. Writer is not safe for concurrent
// use; open one per goroutine.
type Writer struct {
	// onClose is called when the writer is closed.
	onClose func()
	// control stores the control gate held by the virtual writer, and used to track
	// control handoff scenarios with other writers.
	control *control.Gate[*controlResource]
	// wrapError is a function that wraps any error originating from this writer to
	// provide context including the writer's channel key and name.
	wrapError func(error) error
	channel   channel.Channel
	cfg       WriterConfig
	closed    bool
}

// OpenWriter opens a writer on the DB's channel with the given configuration, returning
// the control transfer that occurred as a result of the open. It returns ErrDBClosed if
// the DB is closed, and control.ErrUnauthorized if the writer fails to acquire control
// and ErrOnUnauthorizedOpen is set.
func (db *DB) OpenWriter(cfgs ...WriterConfig) (*Writer, control.Transfer, error) {
	if db.closed.Load() {
		return nil, control.Transfer{}, db.wrapError(ErrDBClosed)
	}
	cfg, err := config.New(WriterConfig{ErrOnUnauthorizedOpen: new(false)}, cfgs...)
	if err != nil {
		return nil, control.Transfer{}, db.wrapError(err)
	}
	w := &Writer{
		cfg:       cfg,
		channel:   db.cfg.Channel,
		wrapError: db.wrapError,
	}
	var transfer control.Transfer
	if w.control, transfer, err = db.controller.OpenGate(control.GateConfig[*controlResource]{
		TimeRange:             cfg.domain(),
		ErrOnUnauthorizedOpen: cfg.ErrOnUnauthorizedOpen,
		Authority:             cfg.Authority,
		Subject:               cfg.Subject,
		OpenResource: func() (*controlResource, error) {
			cr := &controlResource{ck: db.cfg.Channel.Key}
			cr.storeAlignment(telem.NewAlignment(db.leadingAlignment.Add(1), 0))
			return cr, nil
		},
	}); err != nil {
		return nil, control.Transfer{}, db.wrapError(err)
	}
	db.openWriters.Add(1)
	w.onClose = func() { db.openWriters.Add(-1) }
	return w, transfer, nil
}

// Write writes a series to the writer, returning the alignment of the first sample. It
// returns ErrWriterClosed if the writer is closed, a validation error if the series
// data type is incompatible with the channel, and control.ErrUnauthorized if the writer
// does not currently hold control.
func (w *Writer) Write(series telem.Series) (telem.Alignment, error) {
	if w.closed {
		return 0, w.wrapError(ErrWriterClosed)
	}
	if err := w.channel.ValidateSeries(series); err != nil {
		return 0, w.wrapError(err)
	}
	r, err := w.control.Authorize()
	if err != nil {
		return 0, w.wrapError(err)
	}
	// copy the alignment here because we want to return the alignment of the FIRST
	// sample, not the last.
	a := r.loadAlignment()
	r.storeAlignment(a.AddSamples(uint32(series.Len())))
	return a, nil
}

// SetAuthority sets the writer's control authority, returning any control transfer that
// occurred as a result of the change.
func (w *Writer) SetAuthority(authority xcontrol.Authority) control.Transfer {
	return w.control.SetAuthority(authority)
}

// Channel returns the channel being written to.
func (w *Writer) Channel() channel.Channel { return w.channel }

// Close closes the writer and releases its control over the channel, returning any
// control transfer that occurred as a result. Closing an already-closed writer is a
// no-op.
func (w *Writer) Close() (control.Transfer, error) {
	if w.closed {
		return control.Transfer{}, nil
	}
	w.closed = true
	_, t := w.control.Release()
	w.onClose()
	return t, nil
}
