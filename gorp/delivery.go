// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package gorp

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/synnaxlabs/x/observe"
)

// Commit is what one commit of a DB changed.
type Commit struct {
	// Seq is the number of the commit. The first commit of the DB that changes a record
	// is 1, and each later one is one more. A commit that changes no record has none.
	Seq uint64
	// DispatchKey is the key that the caller set with SetDispatchKey, or empty when it
	// set none. The client that made the change uses it to know its own echo.
	DispatchKey string
	// Changes is what the commit changed. The readers of a commit share it, so a reader
	// must not modify it.
	Changes Changes
}

// SetDispatchKey sets the DispatchKey that the Commit of tx carries. Call it before the
// commit, on the goroutine that writes through tx. It panics when tx is a DB.
func SetDispatchKey(tx Tx, key string) {
	state := tx.txIdentity()
	if state == nil {
		panic("[gorp] - SetDispatchKey needs a transaction, not a DB")
	}
	state.dispatchKey = key
}

// Observe returns an observable that delivers each Commit of the DB to its readers in
// commit order, and drops none. The committing goroutine runs the readers after it
// releases the write lock and the store notifies its observers, and before the OnCommit
// hooks run and Commit returns. Readers run under no lock, but each later commit waits
// for them before it delivers.
//
// A reader must not commit a change to the DB on its own goroutine: that commit waits
// for the delivery that runs the reader. The same is true for a key-value observer that
// runs on the committing goroutine.
func (db *DB) Observe() observe.Observable[Commit] { return &db.delivery }

// delivery numbers the commits of a DB and delivers them to readers in commit order.
type delivery struct {
	observer observe.Observer[Commit]
	// connected is the number of readers. A commit reads it under the write lock, where
	// it must not wait for the lock of observer.
	connected atomic.Int64
	// seq is the number of the last commit that took one. The write lock guards it.
	seq uint64
	// tail closes when the last ticket delivers. The write lock guards it.
	tail chan struct{}
}

// ticket is the place of a commit in the delivery order. done is nil when the commit
// delivers nothing.
type ticket struct {
	seq  uint64
	prev chan struct{}
	done chan struct{}
}

func (d *delivery) OnChange(reader func(context.Context, Commit)) observe.Disconnect {
	d.connected.Add(1)
	disconnect := d.observer.OnChange(reader)
	return sync.OnceFunc(func() {
		disconnect()
		d.connected.Add(-1)
	})
}

// take gives the next number to a commit that changed a record. The caller holds the
// write lock.
func (d *delivery) take() ticket {
	d.seq++
	t := ticket{seq: d.seq}
	if d.connected.Load() > 0 {
		t.prev, t.done = d.tail, make(chan struct{})
		d.tail = t.done
	}
	return t
}

// run delivers the transaction of state as the commit of t, after the commit before t
// delivers.
func (d *delivery) run(ctx context.Context, t ticket, state *txState) {
	if t.done == nil {
		return
	}
	defer close(t.done)
	if t.prev != nil {
		<-t.prev
	}
	d.observer.Notify(ctx, Commit{
		Seq:         t.seq,
		DispatchKey: state.dispatchKey,
		Changes:     state.changes.list(),
	})
}
