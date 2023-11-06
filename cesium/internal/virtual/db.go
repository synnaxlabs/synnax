// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package virtual

import (
	"github.com/cockroachdb/errors"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/cesium/internal/controller"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/x/atomic"
	"github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

type controlEntity struct {
	ck    core.ChannelKey
	align telem.Alignment
}

func (e *controlEntity) ChannelKey() core.ChannelKey { return e.ck }

type DB struct {
	Config
	controller  *controller.Controller[*controlEntity]
	openWriters *atomic.Int32Counter
}

type Config struct {
	alamos.Instrumentation
	Channel core.Channel
	FS      fs.FS
}

func Open(cfg Config) (db *DB, err error) {
	if !cfg.Channel.Virtual {
		return nil, errors.Wrap(validate.Error, "channel is not virtual")
	}
	return &DB{
		Config:      cfg,
		controller:  controller.New[*controlEntity](cfg.Channel.Concurrency),
		openWriters: &atomic.Int32Counter{},
	}, nil
}

func (db *DB) LeadingControlState() *controller.State {
	return db.controller.LeadingState()
}

func (db *DB) TryClose() error {
	if db.openWriters.Value() > 0 {
		return errors.New("[unary] channel being written to")
	} else {
		return db.Close()
	}
}

func (db *DB) Close() error { return nil }
