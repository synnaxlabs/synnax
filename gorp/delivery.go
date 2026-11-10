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
	"time"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/observe"
	"go.uber.org/zap"
)

// Commit is what one commit of a DB changed, with the values that callers attached to
// it. Attachment.Of reads the values.
type Commit struct {
	// Seq is the number of the commit. The first commit of the DB that changes a record
	// or carries a value is 1, and each later one is one more. A commit with neither
	// has none.
	Seq uint64
	// Changes is what the commit changed. The readers of a commit share it, so a reader
	// must not modify it.
	Changes  Changes
	attached []attached
}

// Observe returns an observable that delivers each Commit of the DB to its readers in
// commit order, and drops none. The committing goroutine runs the readers after it
// releases the write lock and before Commit returns. Each later commit waits for the
// readers before it delivers.
//
// A reader must not commit a change to the DB on its own goroutine: that commit waits
// for the delivery that runs the reader. Readers run under the read lock of the
// observable, so a reader must not connect or disconnect a reader.
func (db *DB) Observe() observe.Observable[Commit] { return &db.delivery }

// delivery numbers the commits of a DB and delivers them to readers in commit order.
type delivery struct {
	observer observe.Observer[Commit]
	ins      alamos.Instrumentation
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

// take gives the next number to a commit that changed a record or carries a value. The
// caller holds the write lock. The commit joins the delivery order while a reader is
// connected or an earlier commit has not delivered, so a reader that connects in
// between misses no number.
func (d *delivery) take() ticket {
	d.seq++
	t := ticket{seq: d.seq}
	if d.connected.Load() > 0 || d.pending() {
		t.prev, t.done = d.tail, make(chan struct{})
		d.tail = t.done
	}
	return t
}

// pending reports whether the last commit in the delivery order has not delivered. The
// caller holds the write lock.
func (d *delivery) pending() bool {
	if d.tail == nil {
		return false
	}
	select {
	case <-d.tail:
		return false
	default:
		return true
	}
}

// run delivers the transaction of state as the commit of t, after the commit before t
// delivers.
func (d *delivery) run(ctx context.Context, t ticket, state *txState) {
	if t.done == nil {
		return
	}
	defer close(t.done)
	start := time.Now()
	if t.prev != nil {
		<-t.prev
	}
	d.observer.Notify(ctx, Commit{
		Seq:      t.seq,
		Changes:  state.changes.list(),
		attached: state.attached,
	})
	if wait := time.Since(start); wait > waitWarning {
		d.ins.L.Warn(
			"gorp commit waited long for the readers of Observe",
			zap.Duration("wait", wait),
		)
	}
}
