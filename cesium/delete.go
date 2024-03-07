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
	"context"
	"github.com/cockroachdb/errors"
	"github.com/synnaxlabs/x/errutil"
	"github.com/synnaxlabs/x/telem"
	"golang.org/x/sync/semaphore"
	"strconv"
	"sync"
	"time"
)

func (db *DB) DeleteChannel(ch ChannelKey) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	udb, uok := db.unaryDBs[ch]
	if uok {
		if udb.Config.Channel.IsIndex {
			for otherDBKey := range db.unaryDBs {
				if otherDBKey == ch {
					continue
				}
				otherDB := db.unaryDBs[otherDBKey]
				if otherDB.Channel.Index == udb.Config.Channel.Key {
					return errors.New("Could not delete index channel with other channels depending on it")
				}
			}
		}

		if err := udb.TryClose(); err != nil {
			return err
		}
		delete(db.unaryDBs, ch)
		return db.fs.Remove(strconv.Itoa(int(ch)))
	}
	vdb, vok := db.virtualDBs[ch]
	if vok {
		if db.digests.key == ch {
			return errors.New("[cesium] - cannot delete update digest channel")
		}
		if err := vdb.TryClose(); err != nil {
			return err
		}
		delete(db.virtualDBs, ch)
		return db.fs.Remove(strconv.Itoa(int(ch)))
	}

	return ChannelNotFound
}

func (db *DB) DeleteTimeRange(ctx context.Context, ch ChannelKey, tr telem.TimeRange) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	udb, uok := db.unaryDBs[ch]
	if !uok {
		return ChannelNotFound
	}

	// cannot delete an index channel that other channels rely on
	if udb.Config.Channel.IsIndex {
		for otherDBKey := range db.unaryDBs {
			if otherDBKey == ch {
				continue
			}
			otherDB := db.unaryDBs[otherDBKey]
			if otherDB.Channel.Index == udb.Config.Channel.Key && otherDB.Domain.GetBounds().OverlapsWith(tr) {
				return errors.New("Could not delete index channel with other channels depending on it")
			}
		}
	}

	return udb.Delete(ctx, tr)
}

func (db *DB) GarbageCollect(ctx context.Context, maxsizeRead uint32, maxGoRoutine int64) (collected bool, err error) {
	db.mu.RLock()
	defer db.mu.RUnlock()
	sem := semaphore.NewWeighted(maxGoRoutine)
	wg := &sync.WaitGroup{}
	c := errutil.NewCatch(errutil.WithAggregation())

	for _, udb := range db.unaryDBs {
		if err = sem.Acquire(ctx, 1); err != nil {
			return collected, err
		}
		wg.Add(1)
		udb := udb
		go func() {
			defer func() {
				sem.Release(1)
				wg.Done()
			}()
			c.Exec(func() error {
				ok, err := udb.GarbageCollect(ctx, maxsizeRead)
				collected = collected || ok
				return err
			})
		}()
	}
	wg.Wait()
	return collected, c.Error()
}

func (db *DB) AutoGC(ctx context.Context, maxSizeRead uint32, GCInterval time.Duration, maxGoRoutine int64, quit chan struct{}) (collectedTimes int) {
	ticker := time.NewTicker(GCInterval * time.Second)
	defer ticker.Stop()
	collectedTimes = 0

	for {
		select {
		case <-ticker.C:
			collected, err := db.GarbageCollect(ctx, maxSizeRead, maxGoRoutine)
			if err != nil {
				panic(err)
			}
			if collected {
				collectedTimes += 1
			}
		case <-quit:
			return
		}
	}
}
