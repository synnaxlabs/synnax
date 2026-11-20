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
	"sync/atomic"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/math"
)

type counter struct {
	ctr *kv.AtomicInt64Counter
	// durable is the highest value that storage holds. A failed write moves ctr but not
	// durable.
	durable atomic.Int64
	// atOpen is the value that storage held when the counter opened.
	atOpen int64
}

func newCounter(ctx context.Context, db kv.ReadWriter, key []byte) (*counter, error) {
	ctr, err := kv.NewCounter(ctx, db, key)
	if err != nil {
		return nil, err
	}
	c := &counter{ctr: ctr, atOpen: ctr.Value()}
	c.durable.Store(c.atOpen)
	return c, nil
}

// issuedAtOpen reports whether storage held a value of the counter at or past local
// when the counter opened.
func (c *counter) issuedAtOpen(local LocalKey) bool {
	return local != 0 && int64(local) <= c.atOpen
}

// issued reports whether storage holds a value of the counter at or past local, so no
// later add returns local.
func (c *counter) issued(local LocalKey) bool {
	return local != 0 && int64(local) <= c.durable.Load()
}

func (c *counter) add(ctx context.Context, delta LocalKey) (LocalKey, error) {
	if c.ctr.Value()+int64(delta) > int64(math.MaxUint20) {
		return 0, errors.New("maximum number of channels created")
	}
	next, err := c.ctr.Add(ctx, int64(delta))
	if err != nil {
		return 0, err
	}
	for cur := c.durable.Load(); next > cur; cur = c.durable.Load() {
		if c.durable.CompareAndSwap(cur, next) {
			break
		}
	}
	return LocalKey(next), nil
}
