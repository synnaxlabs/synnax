// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package actions

import (
	"context"

	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/lock"
)

// Executor runs action dispatches serialized per entry key, each in its own
// transaction. Two concurrent dispatches on the same entry can never interleave their
// read-reduce-write cycles, so neither can overwrite the other's committed actions.
// Notifications always follow the commit they describe.
type Executor[K comparable, A any] struct {
	db         *gorp.DB
	dispatcher Dispatcher[K, A]
	locks      lock.Keyed[K]
}

// NewExecutor constructs an Executor that opens transactions against db and notifies
// through d.
func NewExecutor[K comparable, A any](
	db *gorp.DB,
	d Dispatcher[K, A],
) *Executor[K, A] {
	return &Executor[K, A]{db: db, dispatcher: d}
}

// Dispatch runs stage inside the key's lock and its own transaction, then notifies the
// actions after the commit. When stage or the commit fails, nothing persists and no
// notification is emitted.
func (e *Executor[K, A]) Dispatch(
	ctx context.Context,
	key K,
	dispatchKey string,
	actions []A,
	stage func(gorp.Tx) error,
) error {
	return e.locks.Do(key, func() error {
		if err := e.db.WithTx(ctx, stage); err != nil {
			return err
		}
		e.dispatcher.Notify(ctx, key, dispatchKey, actions)
		return nil
	})
}
