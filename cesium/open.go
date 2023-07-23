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
	"github.com/samber/lo"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/unary"
	"strconv"
)

func Open(dirname string, opts ...Option) (*DB, error) {
	o := newOptions(dirname, opts...)
	if err := openFS(o); err != nil {
		return nil, err
	}

	o.L.Info("opening cesium time series engine", o.Report().ZapFields()...)

	info, err := o.fs.List()
	if err != nil {
		return nil, err
	}
	_db := &DB{
		options: o,
		dbs:     make(map[core.ChannelKey]unary.DB, len(info)),
		relay:   newRelay(o),
	}
	for _, i := range info {
		key := core.ChannelKey(lo.Must(strconv.Atoi(i.Name())))
		if i.IsDir() && !_db.unaryIsOpen(key) {
			if err = _db.openUnary(Channel{Key: key}); err != nil {
				return nil, err
			}
		}
	}
	return _db, nil
}

func openFS(opts *options) error {
	_fs, err := opts.fs.Sub(opts.dirname)
	opts.fs = _fs
	return err
}
