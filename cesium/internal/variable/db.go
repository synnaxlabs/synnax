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
	"fmt"
	"sync/atomic"

	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/cesium/internal/control"
	"github.com/synnaxlabs/cesium/internal/domain"
	"github.com/synnaxlabs/cesium/internal/index"
	"github.com/synnaxlabs/cesium/internal/meta"
	"github.com/synnaxlabs/cesium/internal/resource"
	xcontrol "github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
)

type DB struct {
	domain           *domain.DB
	controller       *control.Controller[*controlledWriter]
	idx             *index.Domain
	wrapError        func(error) error
	closed           *atomic.Bool
	leadingAlignment *atomic.Uint32
	offsets          *offsetCache
	cfg              Config
}

var ErrDBClosed = resource.NewClosedError("variable.db")

func (db *DB) Channel() channel.Channel { return db.cfg.Channel }

func (db *DB) Index() *index.Domain {
	if !db.cfg.Channel.IsIndex {
		panic(fmt.Sprintf("channel %v is not an index channel", db.cfg.Channel))
	}
	return db.index()
}

func (db *DB) index() *index.Domain {
	if db.idx == nil {
		panic(fmt.Sprintf("channel <%v> index is not set", db.cfg.Channel))
	}
	return db.idx
}

func (db *DB) SetIndex(idx *index.Domain) { db.idx = idx }

func (db *DB) LeadingControlState() *control.State {
	return db.controller.LeadingState()
}

func (db *DB) HasDataFor(ctx context.Context, tr telem.TimeRange) (bool, error) {
	if db.closed.Load() {
		return false, ErrDBClosed
	}
	release, err := db.lockControllerForNonWriteOp(tr, "has_data_for")
	if err != nil {
		return true, errors.Skip(err, xcontrol.ErrUnauthorized)
	}
	defer release()
	hasData, err := db.domain.HasDataFor(ctx, tr)
	return hasData, db.wrapError(err)
}

func (db *DB) Read(ctx context.Context, tr telem.TimeRange) (frame channel.Frame, err error) {
	defer func() { err = db.wrapError(err) }()
	var iter *Iterator
	if iter, err = db.OpenIterator(IterRange(tr)); err != nil {
		return frame, err
	}
	defer func() { err = db.wrapError(iter.Close()) }()
	if !iter.SeekFirst(ctx) {
		return frame, err
	}
	for iter.Next(ctx, telem.TimeSpanMax) {
		frame = frame.Extend(iter.Value())
	}
	return frame, err
}

func (db *DB) Size() telem.Size { return db.domain.Size() }

func (db *DB) Close() error {
	if !db.closed.CompareAndSwap(false, true) {
		return nil
	}
	if err := db.domain.Close(); err != nil {
		if errors.Is(err, resource.ErrOpen) {
			db.closed.Store(false)
		}
		return db.wrapError(err)
	}
	return nil
}

func (db *DB) RenameChannelInMeta(ctx context.Context, newName string) error {
	if db.closed.Load() {
		return ErrDBClosed
	}
	if db.cfg.Channel.Name == newName {
		return nil
	}
	db.cfg.Channel.Name = newName
	return meta.Create(ctx, db.cfg.FS, db.cfg.MetaCodec, db.cfg.Channel)
}

func (db *DB) SetIndexKeyInMeta(ctx context.Context, key channel.Key) error {
	if db.closed.Load() {
		return db.wrapError(ErrDBClosed)
	}
	db.cfg.Channel.Index = key
	return meta.Create(ctx, db.cfg.FS, db.cfg.MetaCodec, db.cfg.Channel)
}

func (db *DB) SetChannelKeyInMeta(ctx context.Context, key channel.Key) error {
	if db.closed.Load() {
		return ErrDBClosed
	}
	db.cfg.Channel.Key = key
	return meta.Create(ctx, db.cfg.FS, db.cfg.MetaCodec, db.cfg.Channel)
}
