// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cesium

import (
	"context"
	"fmt"
	"strconv"
	"sync/atomic"

	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/cesium/internal/meta"
	"github.com/synnaxlabs/cesium/internal/unary"
	"github.com/synnaxlabs/cesium/internal/virtual"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

// Open opens a Cesium database on the specified directory. If the directory is not
// empty, Open attempts to parse its subdirectories into Cesium channels. If any of the
// subdirectories are not in the Cesium format, an error is logged and Open continues
// execution.
func Open(ctx context.Context, dirname string, opts ...Option) (*DB, error) {
	o, err := newOptions(dirname, opts...)
	if err != nil {
		return nil, err
	}
	if err := openFS(o); err != nil {
		return nil, err
	}

	o.L.Debug("opening cesium time series engine", o.Report().ZapFields()...)

	info, err := o.fs.List("")
	if err != nil {
		return nil, err
	}
	db := &DB{options: o, closed: &atomic.Bool{}}
	db.mu.unaryDBs = make(map[channel.Key]unary.DB, len(info))
	db.mu.virtualDBs = make(map[channel.Key]virtual.DB, len(info))
	for _, i := range info {
		if !i.IsDir() {
			db.L.Warn(fmt.Sprintf(
				"found unknown file %s in database root directory",
				i.Name(),
			))
			continue
		}
		key, err := strconv.Atoi(i.Name())
		if err != nil {
			db.L.Error(fmt.Sprintf(
				"failed parsing existing folder <%s> to channel key",
				i.Name()),
				zap.Error(err),
			)
			continue
		}

		if err = db.openVirtualOrUnary(ctx, Channel{Key: ChannelKey(key)}); err != nil {
			return nil, err
		}
	}

	sCtx, cancel := signal.Isolated(signal.WithInstrumentation(o.Instrumentation))
	db.relay = openRelay(sCtx, o.Instrumentation, db.streamingConfig)
	db.startGC(sCtx, o)
	db.shutdown = signal.NewHardShutdown(sCtx, cancel)
	return db, nil
}

func (db *DB) openVirtual(ctx context.Context, ch Channel, fs fs.FS) error {
	if _, isOpen := db.mu.virtualDBs[ch.Key]; isOpen {
		return nil
	}
	v, err := virtual.Open(ctx, virtual.Config{
		MetaCodec:       db.metaCodec,
		FS:              fs,
		Channel:         ch,
		Instrumentation: db.Instrumentation,
	})
	if err != nil {
		return err
	}
	db.mu.virtualDBs[ch.Key] = *v
	return nil
}

func (db *DB) openUnary(ctx context.Context, ch Channel, fs fs.FS) error {
	if _, isOpen := db.mu.unaryDBs[ch.Key]; isOpen {
		return nil
	}
	u, err := unary.Open(ctx, unary.Config{
		FS:              fs,
		MetaCodec:       db.metaCodec,
		Channel:         ch,
		Instrumentation: db.Instrumentation,
		FileSize:        db.fileSize,
		GCThreshold:     db.gcCfg.Threshold,
	})
	if err != nil {
		return err
	}
	// In the case where we index the data using a separate index database, we need to
	// set the index on the unary database. Otherwise, we assume the database is
	// self-indexing.
	if u.Channel().Index != 0 && !u.Channel().IsIndex {
		idxDB, ok := db.mu.unaryDBs[u.Channel().Index]
		if !ok {
			if err = db.openVirtualOrUnary(ctx, Channel{Key: u.Channel().Index}); err != nil {
				return err
			}
			if idxDB, ok = db.mu.unaryDBs[u.Channel().Index]; !ok {
				return validate.PathedError(indexChannelNotFoundError(u.Channel().Index), "index")
			}
		}
		u.SetIndex(idxDB.Index())
	}
	db.mu.unaryDBs[ch.Key] = *u
	return nil
}

func (db *DB) openVirtualOrUnary(ctx context.Context, ch Channel) error {
	fs, err := db.fs.Sub(keyToDirName(ch.Key))
	if err != nil {
		return err
	}
	err = db.openVirtual(ctx, ch, fs)
	if errors.Is(err, virtual.ErrNotVirtual) {
		err = db.openUnary(ctx, ch, fs)
	}
	// For legacy, rate-based channels (V1), attempting to open a unary DB on them will
	// return a meta.ErrIgnoreChannel error, which tells us to just ignore and not open
	// that directory as an actual channel. This is a better alternative to deleting the
	// channel, as we don't want to risk losing user data.
	return errors.Skip(err, meta.ErrIgnoreChannel)
}

func openFS(opts *options) error {
	subFS, err := opts.fs.Sub(opts.dirname)
	opts.fs = subFS
	return err
}
