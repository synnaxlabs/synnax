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
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

// Open opens a Cesium database on the specified directory. If the directory is not
// empty, Open attempts to parse its subdirectories into Cesium channels. If any of the
// subdirectories are not in the Cesium format, an error is logged and Open continues
// execution. ctx covers only the open.
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
		historyChanges: observe.New[HistoryChange](),
	}
	db.mu.dbs.unary = make(map[channel.Key]unary.DB, len(info))
	db.mu.virtualDirs = set.New[channel.Key]()
	db.mu.dbs.hidden = make(map[channel.Key]unary.DB)
	db.mu.skipped = set.New[channel.Key]()
	if db.intents, db.corruptIntents, err = openIntents(o.fs); err != nil {
		return nil, err
	}
	if db.corruptIntents > 0 {
		db.L.Warn(
			"skipped corrupt delete intents, whose data Sweep keeps",
			zap.Int("records", db.corruptIntents),
		)
	}
	var detached []string
	for _, i := range info {
		if i.Name() == intentsFile {
			continue
		}
		if !i.IsDir() {
			db.L.Warn(fmt.Sprintf(
				"found unknown file %s in database root directory",
				i.Name(),
			))
			continue
		}
		// The suffix of a v0.61 or v0.62 removal also carries a random number.
		if strings.Contains(i.Name(), deletedDirSuffix) {
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

		// A delete intends to drop the key, so Sweep decides its fate.
		if db.intents.has(key) {
			db.mu.skipped.Add(key)
			continue
		}
		if err = db.openChannel(ctx, Channel{Key: key}); err != nil {
			return nil, err
		}
	}

	for _, name := range detached {
		if err := db.fs.Remove(name); err != nil {
			db.L.Error(
				"failed to remove deleted channel data",
				zap.String("dir", name),
				zap.Error(err),
			)
		}
	}
	return db, nil
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
		var idxDB unary.DB
		if idxDB, err = db.openIndex(ctx, u.Channel().Index); err != nil {
			return err
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

// openIndex returns the DB of the index channel key, opening it when it is not open.
func (db *DB) openIndex(ctx context.Context, key ChannelKey) (unary.DB, error) {
	if idxDB, ok := db.mu.dbs.unary[key]; ok {
		return idxDB, nil
	}
	if err := db.openChannel(ctx, Channel{Key: key}); err != nil {
		return unary.DB{}, err
	}
	idxDB, ok := db.mu.dbs.unary[key]
	if !ok {
		return unary.DB{}, validate.PathedError(indexChannelNotFoundError(key), "index")
	}
	return idxDB, nil
}

// openChannel opens the channel in the directory of ch.Key. A directory whose metadata
// names a virtual channel holds no storage: openChannel records its key and leaves the
// directory in place.
func (db *DB) openChannel(ctx context.Context, ch Channel) error {
	fs, err := db.fs.Sub(keyToDirName(ch.Key))
	if err != nil {
		return err
	}
	err = db.openUnary(ctx, ch, fs)
	if errors.Is(err, unary.ErrVirtual) {
		db.mu.virtualDirs.Add(ch.Key)
		return nil
	}
	return errors.Skip(err, meta.ErrIgnoreChannel)
}

func openFS(opts *options) error {
	subFS, err := opts.fs.Sub(opts.dirname)
	opts.fs = subFS
	return err
}
