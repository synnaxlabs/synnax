// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package variable

import (
	"context"
	"encoding/binary"
	"sync/atomic"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/cesium/internal/control"
	"github.com/synnaxlabs/cesium/internal/domain"
	"github.com/synnaxlabs/cesium/internal/index"
	"github.com/synnaxlabs/cesium/internal/resource"
	"github.com/synnaxlabs/x/config"
	xcontrol "github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

type WriterConfig struct {
	Persist                  *bool
	EnableAutoCommit         *bool
	ErrOnUnauthorizedOpen    *bool
	Subject                  xcontrol.Subject
	Start                    telem.TimeStamp
	End                      telem.TimeStamp
	AutoIndexPersistInterval telem.TimeSpan
	AlignmentDomainIndex     uint32
	Authority                xcontrol.Authority
}

var (
	_                   config.Config[WriterConfig] = WriterConfig{}
	DefaultWriterConfig                             = WriterConfig{
		Persist:                  new(true),
		EnableAutoCommit:         new(true),
		AutoIndexPersistInterval: 1 * telem.Second,
		ErrOnUnauthorizedOpen:    new(false),
	}
	errWriterClosed = resource.NewClosedError("variable.writer")
)

func (c WriterConfig) Validate() error {
	v := validate.New("variable.writer_config")
	validate.NotEmptyString(v, "subject.key", c.Subject.Key)
	validate.NotNil(v, "err_on_unauthorized_open", c.ErrOnUnauthorizedOpen)
	validate.NotNil(v, "persist", c.Persist)
	validate.NotNil(v, "enable_auto_commit", c.EnableAutoCommit)
	v.Ternary("end", !c.End.IsZero() && c.End.Before(c.Start), "end timestamp must be after or equal to start timestamp")
	return v.Error()
}

func (c WriterConfig) Override(other WriterConfig) WriterConfig {
	c.Start = override.Zero(c.Start, other.Start)
	c.End = override.Zero(c.End, other.End)
	c.Subject = override.If(c.Subject, other.Subject, other.Subject.Key != "")
	c.Authority = override.Numeric(c.Authority, other.Authority)
	c.Persist = override.Nil(c.Persist, other.Persist)
	c.EnableAutoCommit = override.Nil(c.EnableAutoCommit, other.EnableAutoCommit)
	c.AutoIndexPersistInterval = override.Zero(c.AutoIndexPersistInterval, other.AutoIndexPersistInterval)
	c.ErrOnUnauthorizedOpen = override.Nil(c.ErrOnUnauthorizedOpen, other.ErrOnUnauthorizedOpen)
	c.AlignmentDomainIndex = override.Numeric(c.AlignmentDomainIndex, other.AlignmentDomainIndex)
	return c
}

func (c WriterConfig) domain() domain.WriterConfig {
	return domain.WriterConfig{
		Start:                    c.Start,
		End:                      c.End,
		EnableAutoCommit:         c.EnableAutoCommit,
		AutoIndexPersistInterval: c.AutoIndexPersistInterval,
	}
}

func (c WriterConfig) controlTimeRange() telem.TimeRange {
	return c.Start.Range(lo.Ternary(c.End.IsZero(), telem.TimeStampMax, c.End))
}

type controlledWriter struct {
	*domain.Writer
	channelKey channel.Key
	alignment  atomic.Uint64
}

var _ control.Resource = &controlledWriter{}

func (w *controlledWriter) ChannelKey() channel.Key { return w.channelKey }

func (w *controlledWriter) loadAlignment() telem.Alignment {
	return telem.Alignment(w.alignment.Load())
}

func (w *controlledWriter) storeAlignment(a telem.Alignment) {
	w.alignment.Store(uint64(a))
}

type Writer struct {
	control   *control.Gate[*controlledWriter]
	idx       *index.Domain
	wrapError func(error) error
	Channel   channel.Channel
	cfg       WriterConfig
	offsets   *offsetTable
	closed    bool
}

func (db *DB) OpenWriter(ctx context.Context, cfgs ...WriterConfig) (
	w *Writer,
	transfer control.Transfer,
	err error,
) {
	if db.closed.Load() {
		return nil, transfer, db.wrapError(ErrDBClosed)
	}
	cfg, err := config.New(DefaultWriterConfig, cfgs...)
	if err != nil {
		return nil, transfer, err
	}
	w = &Writer{
		cfg:       cfg,
		Channel:   db.cfg.Channel,
		idx:       db.index(),
		wrapError: db.wrapError,
		offsets:   &offsetTable{},
	}
	if w.control, transfer, err = db.controller.OpenGate(control.GateConfig[*controlledWriter]{
		ErrIfControlled:       new(false),
		ErrOnUnauthorizedOpen: cfg.ErrOnUnauthorizedOpen,
		TimeRange:             cfg.controlTimeRange(),
		Authority:             cfg.Authority,
		Subject:               cfg.Subject,
		OpenResource: func() (*controlledWriter, error) {
			dw, err := db.domain.OpenWriter(ctx, cfg.domain())
			cw := &controlledWriter{
				Writer:     dw,
				channelKey: db.cfg.Channel.Key,
			}
			a := telem.NewAlignment(cfg.AlignmentDomainIndex, 0)
			if cfg.AlignmentDomainIndex == 0 {
				a = telem.NewAlignment(db.leadingAlignment.Add(1), 0)
			}
			cw.storeAlignment(a)
			return cw, err
		},
	}); err != nil {
		return nil, transfer, w.wrapError(err)
	}
	return w, transfer, w.wrapError(err)
}

func (w *Writer) sampleCount(dw *domain.Writer) int64 {
	return w.offsets.sampleCount
}

func (w *Writer) Write(series telem.Series) (telem.Alignment, error) {
	if w.closed {
		return 0, w.wrapError(errWriterClosed)
	}
	if err := w.Channel.ValidateSeries(series); err != nil {
		return 0, w.wrapError(err)
	}
	dw, err := w.control.Authorize()
	if err != nil {
		return 0, w.wrapError(err)
	}
	if *w.cfg.Persist {
		baseOffset := uint32(dw.Len())
		w.scanOffsets(series.Data, baseOffset)
		a := telem.NewAlignment(dw.loadAlignment().DomainIndex(), uint32(w.sampleCount(dw.Writer)))
		dw.storeAlignment(a)
		_, err = dw.Write(series.Data)
	} else {
		dw.storeAlignment(dw.loadAlignment().AddSamples(uint32(series.Len())))
	}
	return dw.loadAlignment(), w.wrapError(err)
}

func (w *Writer) scanOffsets(data []byte, baseOffset uint32) {
	offset := 0
	for offset+4 <= len(data) {
		w.offsets.offsets = append(w.offsets.offsets, baseOffset+uint32(offset))
		length := int(binary.LittleEndian.Uint32(data[offset:]))
		offset += 4 + length
		w.offsets.sampleCount++
	}
}

func (w *Writer) DomainIndex() uint32 {
	return w.control.PeekResource().loadAlignment().DomainIndex()
}

func (w *Writer) SetAuthority(a xcontrol.Authority) control.Transfer {
	return w.control.SetAuthority(a)
}

func (w *Writer) Commit(ctx context.Context) (telem.TimeStamp, error) {
	if w.closed {
		return telem.TimeStampMax, w.wrapError(errWriterClosed)
	}
	ts, err := w.commitWithEnd(ctx, telem.TimeStamp(0))
	return ts, w.wrapError(err)
}

func (w *Writer) CommitWithEnd(ctx context.Context, end telem.TimeStamp) (err error) {
	if w.closed {
		return w.wrapError(errWriterClosed)
	}
	_, err = w.commitWithEnd(ctx, end)
	return w.wrapError(err)
}

func (w *Writer) commitWithEnd(ctx context.Context, end telem.TimeStamp) (telem.TimeStamp, error) {
	dw, err := w.control.Authorize()
	if err != nil {
		return 0, err
	}
	if end.IsZero() {
		approx, err := w.idx.Stamp(
			ctx,
			w.cfg.Start,
			w.sampleCount(dw.Writer)-1,
			index.MustBeContinuous,
		)
		if err != nil {
			return 0, err
		}
		if !approx.Exact() {
			return 0, errors.Wrapf(
				validate.ErrValidation,
				"writer start %s cannot be resolved in the index channel %v",
				w.cfg.Start,
				w.idx.Info(),
			)
		}
		end = approx.Lower + 1
	}
	return end, dw.Commit(ctx, end)
}

func (w *Writer) Close() (control.Transfer, error) {
	if w.closed {
		return control.Transfer{}, nil
	}
	w.closed = true
	dw, t := w.control.Release()
	if t.IsRelease() {
		return t, w.wrapError(dw.Close())
	}
	return t, nil
}

func Write(
	ctx context.Context,
	db *DB,
	start telem.TimeStamp,
	series telem.Series,
) (err error) {
	w, _, err := db.OpenWriter(ctx, WriterConfig{
		Start:     start,
		Authority: xcontrol.AuthorityAbsolute,
		Subject:   xcontrol.Subject{Key: uuid.New().String()},
	})
	if err != nil {
		return db.wrapError(err)
	}
	defer func() {
		_, errClose := w.Close()
		err = db.wrapError(errors.Combine(err, errClose))
	}()
	if _, err = w.Write(series); err != nil {
		return err
	}
	_, err = w.Commit(ctx)
	return err
}

func (db *DB) lockControllerForNonWriteOp(tr telem.TimeRange, opName string) (release func(), err error) {
	g, _, err := db.controller.OpenGate(control.GateConfig[*controlledWriter]{
		ErrIfControlled: new(true),
		TimeRange:       tr,
		Authority:       xcontrol.AuthorityAbsolute,
		Subject:         xcontrol.Subject{Key: uuid.NewString(), Name: opName},
		OpenResource: func() (*controlledWriter, error) {
			return &controlledWriter{Writer: nil, channelKey: db.cfg.Channel.Key}, nil
		},
	})
	return func() { g.Release() }, err
}
