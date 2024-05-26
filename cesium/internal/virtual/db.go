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
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/cesium/internal/controller"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
	"sync"
	"sync/atomic"
)

type controlEntity struct {
	ck    core.ChannelKey
	align telem.AlignmentPair
}

func (e *controlEntity) ChannelKey() core.ChannelKey { return e.ck }

type openEntityCount struct {
	sync.RWMutex
	openWriters int
}

func (s *openEntityCount) add(delta int) {
	s.Lock()
	s.openWriters += delta
	s.Unlock()
}

type DB struct {
	Config
	controller *controller.Controller[*controlEntity]
	mu         *openEntityCount
	wrapError  func(error) error
	closed     *atomic.Bool
}

var dbClosed = core.EntityClosed("virtual.db")

type Config struct {
	alamos.Instrumentation
	FS      xfs.FS
	Channel core.Channel
}

func Open(cfg Config) (db *DB, err error) {
	c, err := controller.New[*controlEntity](controller.Config{Concurrency: cfg.Channel.Concurrency, Instrumentation: cfg.Instrumentation})
	if err != nil {
		return nil, err
	}
	return &DB{
		Config:     cfg,
		controller: c,
		wrapError:  core.NewErrorWrapper(cfg.Channel.Key, cfg.Channel.Name),
		mu:         &openEntityCount{},
		closed:     &atomic.Bool{},
	}, nil
}

func (db *DB) LeadingControlState() *controller.State {
	return db.controller.LeadingState()
}

func (db *DB) Close() error {
	if db.closed.Load() {
		return nil
	}
	db.mu.RLock()
	defer db.mu.RUnlock()
	if db.mu.openWriters > 0 {
		return db.wrapError(errors.Newf(fmt.Sprintf("cannot close channel because there are %d unclosed writers accessing it", db.mu.openWriters)))
	}

	db.closed.Swap(true)
	return nil
}
