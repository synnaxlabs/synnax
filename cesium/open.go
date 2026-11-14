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
	"strings"
	"sync/atomic"

	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/cesium/internal/meta"
	"github.com/synnaxlabs/cesium/internal/unary"
	"github.com/synnaxlabs/cesium/internal/virtual"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
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
	db := &DB{
		options:        o,
		closed:         &atomic.Bool{},
		controlUpdates: observe.New[ControlUpdate](),
		historyChanges: observe.New[HistoryChange](),
	}
	db.mu.dbs.unary = make(map[channel.Key]unary.DB, len(info))
	db.mu.dbs.virtual = make(map[channel.Key]virtual.DB, len(info))
	var detached []string
	for _, i := range info {
		if !i.IsDir() {
			db.L.Warn(fmt.Sprintf(
				"found unknown file %s in database root directory",
				i.Name(),
			))
			continue
		}
		if strings.Contains(i.Name(), deletedDirInfix) {
			detached = append(detached, i.Name())
			continue
		}
		key, err := dirNameToKey(i.Name())
		if err != nil {
			db.L.Error(fmt.Sprintf(
				"failed parsing existing folder <%s> to channel key",
				i.Name(),
			),
				zap.Error(err),
			)
			continue
		}

		if err = db.openVirtualOrUnary(ctx, Channel{Key: key}); err != nil {
			return nil, err
		}
	}

	sCtx, cancel := signal.Isolated(signal.WithInstrumentation(o.Instrumentation))
	db.relay = openRelay(sCtx, o)
	db.startGC(sCtx, o)
	db.shutdown = signal.NewHardShutdown(sCtx, cancel)
	db.removeDirs(detached...)
	return db, nil
}

func (db *DB) openVirtual(ctx context.Context, ch Channel, fs fs.FS) error {
	if _, isOpen := db.mu.dbs.virtual[ch.Key]; isOpen {
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
	db.mu.dbs.virtual[ch.Key] = *v
	return nil
}

func (db *DB) openUnary(ctx context.Context, ch Channel, fs fs.FS) (err error) {
	if _, isOpen := db.mu.dbs.unary[ch.Key]; isOpen {
		return nil
	}
	u, err := unary.Open(ctx, unary.Config{
		FS:              fs,
		MetaCodec:       db.metaCodec,
		Channel:         ch,
		Instrumentation: db.Instrumentation,
		FileSize:        db.fileSize,
		GCThreshold:     db.gcCfg.Threshold,
		Clock:           db.clock,
		OnHistoryChange: func(ctx context.Context, tr telem.TimeRange) {
			db.historyChanges.Notify(ctx, HistoryChange{Key: ch.Key, TimeRange: tr})
		},
	})
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			err = errors.Combine(err, u.Close())
		}
	}()
	// For non-index channels, resolve the index DB. The index DB is always a
	// fixed-density unary DB (index channels are TimestampT with IsIndex=true).
	if u.Channel().Index != 0 && !u.Channel().IsIndex {
		idxDB, ok := db.mu.dbs.unary[u.Channel().Index]
		if !ok {
			if err = db.openVirtualOrUnary(
				ctx,
				Channel{Key: u.Channel().Index},
			); err != nil {
				return err
			}
			if idxDB, ok = db.mu.dbs.unary[u.Channel().Index]; !ok {
				return validate.PathedError(
					indexChannelNotFoundError(u.Channel().Index),
					"index",
				)
			}
		}
		u.SetIndex(idxDB.Index())
		// A failed truncation does not stop the DB from opening. The channel opens
		// untruncated, as a crash leaves it, or cut in memory when only the save of
		// the cut failed.
		if truncErr := u.TruncateToIndex(ctx); truncErr != nil {
			db.L.Error(
				"failed to truncate a channel to the samples its index has timestamps for",
				zap.Error(truncErr),
			)
		}
	}
	db.mu.dbs.unary[ch.Key] = *u
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
	return errors.Skip(err, meta.ErrIgnoreChannel)
}

func openFS(opts *options) error {
	subFS, err := opts.fs.Sub(opts.dirname)
	opts.fs = subFS
	return err
}
