// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package actions provides shared runtime primitives for action-based dispatch on
// stateful services in Synnax. The schematic, table, and line plot services all
// implement the same shape: a discriminated union of actions reduced atomically
// against a stored entry, then notified to subscribers so the cluster signals
// subsystem can broadcast the sequence on a free channel. This package hosts the
// pieces of that shape that are independent of the per-service action variant
// set: the Scoped envelope sent to subscribers, the DispatchRequest wire body
// accepted by the API layer, the State that owns the observer and the monotonic
// sequence counter, the Dispatcher that stamps and forwards each notification,
// and the PublishSignals helper that wires the set and delete pipelines onto
// signals channels.
package actions

import (
	"context"
	"sync/atomic"

	"github.com/synnaxlabs/x/observe"
)

// Scoped wraps an action sequence with the targeted entry key, the
// dispatch-batch identifier supplied by the originating client, and a
// monotonic sequence number assigned by the node that handled the dispatch.
// Subscribers compare Seq against the last value they applied for the same
// Key to decide whether the frame is fresh or a stale echo. DispatchKey is
// carried verbatim so the originating client can match the echo against the
// set of outstanding local replays it registered before sending, and skip a
// redundant reduce when no foreign action interleaved.
//
// Seq is monotonic per originating node: in multi-node deployments two nodes
// may emit overlapping Seq values for the same Key, so cross-node ordering is
// best-effort until a cluster-wide ordering primitive lands as part of the
// broader server-side undo work.
type Scoped[K comparable, A any] struct {
	// Key identifies the entry the action sequence targets.
	Key K `json:"key" msgpack:"key"`
	// DispatchKey is the client-generated batch identifier carried verbatim
	// from the originating dispatch onto every echo.
	DispatchKey string `json:"dispatch_key" msgpack:"dispatch_key"`
	// Seq is the per-node monotonic sequence number stamped by the Dispatcher
	// after the underlying transaction commits.
	Seq uint64 `json:"seq" msgpack:"seq"`
	// Actions is the action sequence applied in order. Subscribers receive the
	// sequence verbatim; reducing it is the consumer's responsibility.
	Actions []A `json:"actions" msgpack:"actions"`
}

// DispatchRequest is the wire-format request body for the per-service dispatch
// endpoint. The API layer decodes it, enforces access, and forwards Key,
// DispatchKey, and Actions to the service-level Writer.Dispatch. The shape
// matches Scoped on the fields a client controls: the Seq is server-assigned
// and therefore not part of the request.
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

// State owns the per-service observer and monotonic sequence counter used by
// every Writer the service issues. Construct it once in OpenService, hand
// Dispatcher() to each Writer, and expose subscriptions via OnAction. The
// observer is always non-nil so writers do not need to nil-check it.
//
// Seq is in-memory and resets on process restart. Clients reload service state
// on reconnect and reset their per-key high-water marks alongside it, so the
// reset is observable but not destructive. In multi-node deployments each node
// has its own counter, making cross-node ordering best-effort.
type State[K comparable, A any] struct {
	observer observe.Observer[Scoped[K, A]]
	seq      atomic.Uint64
}

// NewState constructs a State with a fresh observer and a zero sequence counter.
func NewState[K comparable, A any]() *State[K, A] {
	return &State[K, A]{observer: observe.New[Scoped[K, A]]()}
}

// OnAction subscribes the given handler to the action stream emitted by every
// Dispatcher bound to this State. The handler runs synchronously inside the
// Dispatcher.Notify call. The returned Disconnect removes the handler.
func (s *State[K, A]) OnAction(
	handler func(context.Context, Scoped[K, A]),
) observe.Disconnect {
	return s.observer.OnChange(handler)
}

// Dispatcher returns a value-typed Dispatcher bound to this State. Embed the
// returned value on a Writer; each call to Notify increments the shared
// sequence counter and fans the resulting Scoped envelope out to every handler
// registered via OnAction.
func (s *State[K, A]) Dispatcher() Dispatcher[K, A] {
	return Dispatcher[K, A]{observer: s.observer, seq: &s.seq}
}

// Dispatcher stamps an incrementing Seq onto each action sequence and forwards
// the resulting Scoped envelope to the State's observer. Construct via
// State.Dispatcher; the zero value is not usable.
type Dispatcher[K comparable, A any] struct {
	observer observe.Observer[Scoped[K, A]]
	seq      *atomic.Uint64
}

// Notify increments the bound sequence counter and emits a Scoped envelope to
// the bound observer. Call Notify only after the underlying transaction
// commits so subscribers never observe an action sequence whose effects are
// not yet persisted.
func (d Dispatcher[K, A]) Notify(
	ctx context.Context,
	key K,
	dispatchKey string,
	actions []A,
) {
	d.observer.Notify(ctx, Scoped[K, A]{
		Key:         key,
		DispatchKey: dispatchKey,
		Seq:         d.seq.Add(1),
		Actions:     actions,
	})
}
