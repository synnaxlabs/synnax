// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel

import (
	"context"
	"encoding/binary"
	"slices"
	"sync"
	"sync/atomic"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/math"
	"github.com/synnaxlabs/x/query"
)

// firstOpenSuffix names the record of the value that storage held at the first open of
// a counter.
const firstOpenSuffix = ".firstOpen"

type counter struct {
	// mu serializes adds, as two concurrent adds of ctr can write their values out of
	// order.
	mu  sync.Mutex
	ctr *kv.AtomicInt64Counter
	// durable is the value that storage holds. A failed write moves ctr but not
	// durable.
	durable atomic.Int64
	// atFirstOpen is the value that storage held at the first open of the counter.
	atFirstOpen int64
}

func newCounter(ctx context.Context, db kv.ReadWriter, key []byte) (*counter, error) {
	ctr, err := kv.NewCounter(ctx, db, key)
	if err != nil {
		return nil, err
	}
	c := &counter{ctr: ctr}
	c.durable.Store(ctr.Value())
	if c.atFirstOpen, err = firstOpenValue(ctx, db, key, ctr.Value()); err != nil {
		return nil, err
	}
	return c, nil
}

// firstOpenValue returns the value that storage held at the first open of the counter
// at key. On the first open, it records value as that value.
func firstOpenValue(
	ctx context.Context,
	db kv.ReadWriter,
	key []byte,
	value int64,
) (int64, error) {
	record := append(slices.Clone(key), firstOpenSuffix...)
	b, closer, err := db.Get(record)
	if err == nil {
		first := int64(binary.LittleEndian.Uint64(b))
		return first, closer.Close()
	}
	if !errors.Is(err, query.ErrNotFound) {
		return 0, err
	}
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], uint64(value))
	return value, db.Set(ctx, record, buf[:])
}

// issuedAtFirstOpen reports whether storage held a value of the counter at or past
// local at the first open of the counter.
func (c *counter) issuedAtFirstOpen(local LocalKey) bool {
	return local != 0 && int64(local) <= c.atFirstOpen
}

// issued reports whether storage holds a value of the counter at or past local, so no
// later add returns local.
func (c *counter) issued(local LocalKey) bool {
	return local != 0 && int64(local) <= c.durable.Load()
}

func (c *counter) add(ctx context.Context, delta LocalKey) (LocalKey, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.ctr.Value()+int64(delta) > int64(math.MaxUint20) {
		return 0, errors.New("maximum number of channels created")
	}
	next, err := c.ctr.Add(ctx, int64(delta))
	if err != nil {
		return 0, err
	}
	c.durable.Store(next)
	return LocalKey(next), nil
}
