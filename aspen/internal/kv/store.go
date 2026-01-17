// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package kv

import (
	"context"
	"maps"

	"github.com/synnaxlabs/x/confluence"
	xstore "github.com/synnaxlabs/x/store"
)

type storeState map[string]Operation

func (s storeState) Copy() storeState {
	mCopy := make(storeState, len(s))
	maps.Copy(mCopy, s)
	return mCopy
}

func (s storeState) toBatchRequest(ctx context.Context) TxRequest {
	b := TxRequest{Context: ctx, Operations: make([]Operation, 0, len(s))}
	for _, op := range s {
		if op.state != gossipStateInfected {
			continue
		}
		// Since we're not writing to any underlying storage, any error
		// should panic.
		b.Operations = append(b.Operations, op)
	}
	return b
}

type store xstore.Store[storeState]

func newStore() store {
	return xstore.New(func(m storeState) storeState { return m.Copy() })
}

type storeEmitter struct {
	store store
	confluence.Emitter[TxRequest]
}

func newStoreEmitter(s store, cfg Config) source {
	se := &storeEmitter{store: s}
	se.Interval = cfg.GossipInterval
	se.Emitter.Emit = se.Emit
	return se
}

func (e *storeEmitter) Emit(ctx context.Context) (TxRequest, error) {
	s, release := e.store.PeekState()
	defer release()
	return s.toBatchRequest(ctx), nil
}

type storeSink struct {
	confluence.UnarySink[TxRequest]
	store store
}

func newStoreSink(s store) sink {
	ss := &storeSink{store: s}
	ss.Sink = ss.Store
	return ss
}

func (s *storeSink) Store(ctx context.Context, br TxRequest) error {
	snap := s.store.CopyState()
	for _, op := range br.Operations {
		snap[string(op.Key)] = op
	}
	s.store.SetState(ctx, snap)
	return nil
}
