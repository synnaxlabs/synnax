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
	"github.com/synnaxlabs/cesium/internal/claim"
	"github.com/synnaxlabs/cesium/internal/resource"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/telem"
)

var ErrWriterClosed = resource.NewClosedError("virtual.writer")

type WriterConfig struct {
	Start telem.TimeStamp
	End   telem.TimeStamp
}

var _ config.Config[WriterConfig] = WriterConfig{}

func (cfg WriterConfig) Validate() error { return nil }

func (cfg WriterConfig) Override(other WriterConfig) WriterConfig {
	cfg.Start = override.Zero(cfg.Start, other.Start)
	cfg.End = override.Zero(cfg.End, other.End)
	return cfg
}

func (cfg WriterConfig) domain() telem.TimeRange {
	return telem.TimeRange{
		Start: cfg.Start,
		End:   lo.Ternary(cfg.End.IsZero(), telem.TimeStampMax, cfg.End),
	}
}

type Writer struct {
	// onClose is called when the writer is closed.
	onClose func()
	// claim is the writer's claim on the time range it writes.
	claim *claim.Claim[*cursor]
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

func (db *DB) OpenWriter(_ context.Context, cfgs ...WriterConfig) (*Writer, error) {
	if db.closed.Load() {
		return nil, db.wrapError(ErrDBClosed)
	}
	cfg, err := config.New(WriterConfig{}, cfgs...)
	if err != nil {
		return nil, db.wrapError(err)
	}
	w := &Writer{
		WriterConfig: cfg,
		Channel:      db.cfg.Channel,
		wrapError:    db.wrapError,
	}
	if w.claim, err = db.claims.Open(claim.Config[*cursor]{
		TimeRange: cfg.domain(),
		Open: func() (*cursor, error) {
			a, err := db.leadingAlignment.Next()
			if err != nil {
				return nil, err
			}
			c := &cursor{}
			c.storeAlignment(a)
			return c, nil
		},
	}); err != nil {
		return nil, db.wrapError(err)
	}
	db.openWriters.Add(1)
	w.onClose = func() {
		db.openWriters.Add(-1)
	}
	return w, nil
}

func (w *Writer) Write(series telem.Series) (telem.Alignment, error) {
	if w.closed {
		return 0, w.wrapError(ErrWriterClosed)
	}
	if err := w.Channel.ValidateSeries(series); err != nil {
		return 0, w.wrapError(err)
	}
	var a telem.Alignment
	if err := w.claim.Use(func(e *cursor) error {
		// copy the alignment here because we want to return the alignment of the
		// FIRST sample, not the last.
		a = e.loadAlignment()
		e.storeAlignment(a.AddSamples(uint32(series.Len())))
		return nil
	}); err != nil {
		return 0, w.wrapError(err)
	}
	return a, nil
}

func (w *Writer) Close() error {
	if w.closed {
		return nil
	}
	w.closed = true
	w.claim.Release()
	w.onClose()
	return nil
}
