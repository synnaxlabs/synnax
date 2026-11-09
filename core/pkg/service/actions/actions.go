// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package actions provides shared runtime primitives for action-based dispatch on
// stateful services in Synnax. Each such service implements the same shape: a
// discriminated union of actions reduced atomically against a stored entry, then
// attached to the transaction that applied them, so the commit delivers them to the
// change channels. This package hosts the pieces of that shape that are independent of
// the per-service action variant set: the Scoped envelope that a commit delivers, the
// DispatchRequest wire body accepted by the API layer, the State that owns the
// attachment, and the Dispatcher that attaches each batch.
package actions

import (
	"context"
	"sync/atomic"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/validate"
)

// ErrCoreOnly rejects a client dispatch of an action that only the Core emits.
var ErrCoreOnly = errors.Wrap(
	validate.ErrValidation,
	"only the Core dispatches the action",
)

// Scoped wraps an action sequence with the targeted entry key, the dispatch-batch
// identifier supplied by the originating client, and the number of the batch.
// Subscribers compare Seq against the last value they applied for the same
// Key to decide whether the frame is fresh or a stale echo. DispatchKey is carried
// verbatim so the originating client can match the echo against the set of outstanding
// local replays it registered before sending, and skip a redundant reduce when no
// foreign action interleaved.
type Scoped[K comparable, A any] struct {
	// Key identifies the entry the action sequence targets.
	Key K `json:"key" msgpack:"key"`
	// DispatchKey is the client-generated batch identifier carried verbatim
	// from the originating dispatch onto every echo.
	DispatchKey string `json:"dispatch_key" msgpack:"dispatch_key"`
	// Seq is the number of the batch. A batch takes a higher number than every batch
	// that an earlier commit or the same commit attached before it, so Seq rises for
	// each Key. Numbers that an aborted transaction took are skipped.
	Seq uint64 `json:"seq" msgpack:"seq"`
	// Actions is the action sequence applied in order. Subscribers receive the
	// sequence verbatim; reducing it is the consumer's responsibility.
	Actions []A `json:"actions" msgpack:"actions"`
}

// DispatchRequest is the wire-format request body for the per-service dispatch
// endpoint. The API layer decodes it and forwards Key, DispatchKey, and Actions to the
// service-level Writer.Apply. The shape matches Scoped on the fields a client controls:
// the Seq is server-assigned and therefore not part of the request.
type DispatchRequest[K comparable, A any] struct {
	// Key identifies the entry the action sequence targets.
	Key K `json:"key" msgpack:"key"`
	// DispatchKey is a client-generated batch identifier. It is carried
	// verbatim onto the resulting Scoped envelope so the originator can
	// recognize its own echo.
	DispatchKey string `json:"dispatch_key" msgpack:"dispatch_key"`
	// Actions is the action sequence to apply.
	Actions []A `json:"actions" msgpack:"actions"`
}

// DispatchResponse is the wire-format response body for a dispatch endpoint whose
// service adds actions to a dispatch, such as a set of a field that only the Core
// computes.
type DispatchResponse[A any] struct {
	// Actions are the actions the Core added after the dispatched ones.
	Actions []A `json:"actions" msgpack:"actions"`
}

// State attaches the action batches of one service to the transactions that apply
// them. Construct it once in OpenService and hand Dispatcher() to each Writer.
type State[K comparable, A any] struct {
	db       *gorp.DB
	attached *gorp.Attachment[Scoped[K, A]]
	// seq is the number of the last batch attached. A batch attaches under the write
	// lock of db, so the numbers rise in commit order.
	seq atomic.Uint64
}

// NewState constructs a State on db.
func NewState[K comparable, A any](db *gorp.DB) *State[K, A] {
	return &State[K, A]{db: db, attached: gorp.NewAttachment[Scoped[K, A]]()}
}

// Of returns the action batches that the transaction of c attached, in the order that
// it attached them. The readers of c share the batches, so a reader must not modify
// them.
func (s *State[K, A]) Of(c gorp.Commit) []Scoped[K, A] { return s.attached.Of(c) }

// OnChange calls handler with each batch that Of returns for each commit of the DB. The
// handler runs on the committing goroutine, as the readers of gorp.DB.Observe do. The
// returned Disconnect removes the handler.
func (s *State[K, A]) OnChange(
	handler func(context.Context, Scoped[K, A]),
) observe.Disconnect {
	return s.db.Observe().OnChange(func(ctx context.Context, c gorp.Commit) {
		for _, batch := range s.Of(c) {
			handler(ctx, batch)
		}
	})
}

// Dispatcher returns a Dispatcher that attaches batches for the State.
func (s *State[K, A]) Dispatcher() Dispatcher[K, A] {
	return Dispatcher[K, A]{state: s}
}

// Dispatcher attaches action batches to transactions. Construct it via
// State.Dispatcher; the zero value is not usable.
type Dispatcher[K comparable, A any] struct {
	state       *State[K, A]
	dispatchKey string
}

// WithDispatchKey returns a copy of the Dispatcher that stamps dispatchKey onto each
// batch it attaches, so the client that sent the write recognizes its own echo.
func (d Dispatcher[K, A]) WithDispatchKey(dispatchKey string) Dispatcher[K, A] {
	d.dispatchKey = dispatchKey
	return d
}

// Attach attaches actions on the entry with key to tx as one batch with the next
// number, after the batches attached to tx before. The commit of tx delivers them. It
// panics when tx is a DB.
func (d Dispatcher[K, A]) Attach(tx gorp.Tx, key K, actions ...A) {
	d.state.attached.Add(tx, Scoped[K, A]{
		Key:         key,
		DispatchKey: d.dispatchKey,
		Seq:         d.state.seq.Add(1),
		Actions:     actions,
	})
}
