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
	"fmt"
	"github.com/cockroachdb/errors"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/cesium/internal/controller"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"sync"
)

type controlEntity struct {
	ck    core.ChannelKey
	align telem.Alignment
}

func (e *controlEntity) ChannelKey() core.ChannelKey { return e.ck }

type openEntityCount struct {
	sync.RWMutex
	openWriters int
}

func (c *openEntityCount) Add(delta int) {
	c.Lock()
	c.openWriters += delta
	c.Unlock()
}

type DB struct {
	Config
	controller *controller.Controller[*controlEntity]
	mu         *openEntityCount
}

type Config struct {
	alamos.Instrumentation
	Channel core.Channel
}

func Open(cfg Config) (db *DB, err error) {
	if !cfg.Channel.Virtual {
		return nil, errors.Wrap(validate.Error, "channel is not virtual")
	}
	return &DB{
		Config:     cfg,
		controller: controller.New[*controlEntity](cfg.Channel.Concurrency),
		mu:         &openEntityCount{},
	}, nil
}

func (db *DB) LeadingControlState() *controller.State {
	return db.controller.LeadingState()
}

func (db *DB) TryClose() error {
	db.mu.RLock()
	defer db.mu.RUnlock()
	if db.mu.openWriters > 0 {
		return errors.Newf(fmt.Sprintf("[cesium] - cannot close channel because there are currently %d unclosed writers accessing it", db.mu.openWriters))
	}
	return db.Close()
}

func (db *DB) Close() error { return nil }
