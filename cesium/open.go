// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cesium

import (
	"fmt"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/meta"
	"github.com/synnaxlabs/cesium/internal/unary"
	"github.com/synnaxlabs/cesium/internal/version"
	"github.com/synnaxlabs/cesium/internal/virtual"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
	"strconv"
	"sync"
	"sync/atomic"
)

func Open(dirname string, opts ...Option) (*DB, error) {
	o := newOptions(dirname, opts...)
	if err := openFS(o); err != nil {
		return nil, err
	}

	o.L.Info("opening cesium time series engine", o.Report().ZapFields()...)

	sCtx, cancel := signal.Isolated(signal.WithInstrumentation(o.Instrumentation))

	info, err := o.fs.List("")
	if err != nil {
		return nil, err
	}
	db := &DB{
		options:    o,
		mu:         sync.RWMutex{},
		unaryDBs:   make(map[core.ChannelKey]unary.DB, len(info)),
		virtualDBs: make(map[core.ChannelKey]virtual.DB, len(info)),
		relay:      newRelay(sCtx),
		closed:     &atomic.Bool{},
		shutdown:   signal.NewShutdown(sCtx, cancel),
	}
	for _, i := range info {
		if i.IsDir() {
			key, err := strconv.Atoi(i.Name())
			if err != nil {
				db.options.L.Error(fmt.Sprintf("failed parsing existing folder <%s> to channel key", i.Name()), zap.Error(err))
				continue
			}

			if err = db.openVirtualOrUnary(Channel{Key: ChannelKey(key)}); err != nil {
				return nil, err
			}
		} else {
			db.options.L.Warn(fmt.Sprintf("Found unknown file %s in database root directory", i.Name()))
		}
	}

	db.startGC(sCtx, o)

	return db, nil
}

func (db *DB) openVirtualOrUnary(ch Channel) error {
	if db.closed.Load() {
		return errDBClosed
	}

	db.mu.Lock()
	defer db.mu.Unlock()
	fs, err := db.fs.Sub(strconv.Itoa(int(ch.Key)))
	if err != nil {
		return err
	}
	ch, err = meta.ReadOrCreate(fs, ch, db.metaECD)
	if err != nil {
		return err
	}
	if ch.Virtual {
		_, isOpen := db.virtualDBs[ch.Key]
		if isOpen {
			return nil
		}
		if ch.Version != version.CurrentVersion {
			ch.Version = version.CurrentVersion
			err = meta.Create(fs, db.metaECD, ch)
			if err != nil {
				return err
			}
		}
		v, err := virtual.Open(virtual.Config{FS: fs, Channel: ch, Instrumentation: db.options.Instrumentation})
		if err != nil {
			return err
		}
		db.virtualDBs[ch.Key] = *v
	} else {
		_, isOpen := db.unaryDBs[ch.Key]
		if isOpen {
			return nil
		}
		if ch.Version != version.CurrentVersion {
			err = version.Migrate(fs, ch.Version, version.CurrentVersion)
			if err != nil {
				return err
			}

			ch.Version = version.CurrentVersion
			err = meta.Create(fs, db.metaECD, ch)
			if err != nil {
				return err
			}
		}
		u, err := unary.Open(unary.Config{FS: fs, Channel: ch, Instrumentation: db.options.Instrumentation, FileSize: db.options.fileSize, GCThreshold: db.options.gcCfg.GCThreshold})
		if err != nil {
			return err
		}
		// In the case where we index the data using a separate index database, we
		// need to set the index on the unary database. Otherwise, we assume the database
		// is self-indexing.
		if u.Channel.Index != 0 && !u.Channel.IsIndex {
			idxDB, ok := db.unaryDBs[u.Channel.Index]
			if !ok {
				err = db.openVirtualOrUnary(Channel{Key: u.Channel.Index})
				if err != nil {
					return err
				}
				idxDB, ok = db.unaryDBs[u.Channel.Index]
				if !ok {
					return validate.FieldError{Field: "index", Message: fmt.Sprintf("index channel <%v> does not exist", u.Channel.Index)}
				}
			}
			u.SetIndex((&idxDB).Index())
		}
		db.unaryDBs[ch.Key] = *u
	}
	return nil
}

func openFS(opts *options) error {
	_fs, err := opts.fs.Sub(opts.dirname)
	opts.fs = _fs
	return err
}
