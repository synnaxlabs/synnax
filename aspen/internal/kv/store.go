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
	"sync"

	"github.com/synnaxlabs/x/confluence"
)

type storeState map[string]Operation

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

// kvStore manages the storeState map with its own mutex. apply() mutates entries
// in-place under the write lock, avoiding the full map copy that the previous
// CopyState()+SetState() pattern incurred on every write.
type kvStore struct {
	mu   sync.RWMutex
	data storeState
}

func newStore() *kvStore {
	return &kvStore{data: make(storeState)}
}

func (s *kvStore) PeekState() (storeState, func()) {
	s.mu.RLock()
	return s.data, s.mu.RUnlock
}

// apply writes operations directly into the map under the write lock.
func (s *kvStore) apply(ops []Operation) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, op := range ops {
		s.data[string(op.Key)] = op
	}
}

type storeEmitter struct {
	store *kvStore
	confluence.Emitter[TxRequest]
}

func newStoreEmitter(s *kvStore, cfg Config) source {
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
	store *kvStore
}

func newStoreSink(s *kvStore) sink {
	ss := &storeSink{store: s}
	ss.Sink = ss.Store
	return ss
}

func (s *storeSink) Store(_ context.Context, br TxRequest) error {
	s.store.apply(br.Operations)
	return nil
}
