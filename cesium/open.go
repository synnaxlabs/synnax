// Copyright 2025 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/cesium/internal/unary"
	"github.com/synnaxlabs/cesium/internal/virtual"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
	"strconv"
	"sync/atomic"
)

// Open opens a Cesium database on the specified directory. If the directory is not
// empty, Open attempts to parse its subdirectories into Cesium channels. If any of
// the subdirectories are not in the Cesium format, an error is logged and Open continues
// execution.
func Open(dirname string, opts ...Option) (*DB, error) {
	o := newOptions(dirname, opts...)
	if err := openFS(o); err != nil {
		return nil, err
	}

	o.L.Debug("opening cesium time series engine", o.Report().ZapFields()...)

	info, err := o.fs.List("")
	if err != nil {
		return nil, err
	}
	db := &DB{
		options:    o,
		unaryDBs:   make(map[core.ChannelKey]unary.DB, len(info)),
		virtualDBs: make(map[core.ChannelKey]virtual.DB, len(info)),
		closed:     &atomic.Bool{},
	}
	for _, i := range info {
		if i.IsDir() {
			key, err := strconv.Atoi(i.Name())
			if err != nil {
				db.options.L.Error(fmt.Sprintf(
					"failed parsing existing folder <%s> to channel key",
					i.Name()),
					zap.Error(err),
				)
				continue
			}

			if err = db.openVirtualOrUnary(Channel{Key: ChannelKey(key)}); err != nil {
				return nil, err
			}
		} else {
			db.options.L.Warn(fmt.Sprintf(
				"Found unknown file %s in database root directory",
				i.Name(),
			))
		}
	}

	sCtx, cancel := signal.Isolated(signal.WithInstrumentation(o.Instrumentation))
	db.relay = openRelay(sCtx)
	db.startGC(sCtx, o)
	db.shutdown = signal.NewShutdown(sCtx, cancel)
	return db, nil
}

func (db *DB) openVirtual(ch Channel, fs xfs.FS) error {
	_, isOpen := db.virtualDBs[ch.Key]
	if isOpen {
		return nil
	}
	v, err := virtual.Open(virtual.Config{
		MetaCodec:       db.metaCodec,
		FS:              fs,
		Channel:         ch,
		Instrumentation: db.options.Instrumentation,
	})
	if err != nil {
		return err
	}
	db.virtualDBs[ch.Key] = *v
	return nil
}

func (db *DB) openUnary(ch Channel, fs xfs.FS) error {
	_, isOpen := db.unaryDBs[ch.Key]
	if isOpen {
		return nil
	}
	u, err := unary.Open(unary.Config{
		FS:              fs,
		MetaCodec:       db.metaCodec,
		Channel:         ch,
		Instrumentation: db.options.Instrumentation,
		FileSize:        db.options.fileSize,
		GCThreshold:     db.options.gcCfg.GCThreshold,
	})
	if err != nil {
		return err
	}
	// In the case where we index the data using a separate index database, we
	// need to set the index on the unary database. Otherwise, we assume the database
	// is self-indexing.
	if u.Channel().Index != 0 && !u.Channel().IsIndex {
		idxDB, ok := db.unaryDBs[u.Channel().Index]
		if ok {
			u.SetIndex(idxDB.Index())
		}
		err = db.openVirtualOrUnary(Channel{Key: u.Channel().Index})
		if err != nil {
			return err
		}
		idxDB, ok = db.unaryDBs[u.Channel().Index]
		if !ok {
			return validate.FieldError{Field: "index", Message: fmt.Sprintf("index channel <%v> does not exist", u.Channel().Index)}
		}
	}
	db.unaryDBs[ch.Key] = *u
	return nil
}

func (db *DB) openVirtualOrUnary(ch Channel) error {
	fs, err := db.fs.Sub(strconv.Itoa(int(ch.Key)))
	if err != nil {
		return err
	}
	err = db.openVirtual(ch, fs)
	if errors.Is(err, virtual.ErrNotVirtual) {
		return db.openUnary(ch, fs)
	}
	return err
}

func openFS(opts *options) error {
	_fs, err := opts.fs.Sub(opts.dirname)
	opts.fs = _fs
	return err
}
