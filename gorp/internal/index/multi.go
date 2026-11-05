// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package index

import (
	"context"
	"slices"
	"sync"

	"github.com/synnaxlabs/x/set"
)

// Multi is an exact-match index from values of type V to the primary keys of type K
// that hold them. Unlike Lookup, one key can hold any number of values. Multi is safe
// for concurrent use.
type Multi[S Scope, K, V comparable] struct {
	// mu guards forward, reverse, and population.err. Populate holds it for writing
	// until the load finishes.
	mu         sync.RWMutex
	population population
	forward    map[V]holders[K]
	reverse    map[K][]V
	overlay    overlay[S, *delta[K, V]]
	announcements[K, V]
}

// NewMulti returns an empty Multi that waits for its bulk load.
func NewMulti[S Scope, K, V comparable]() *Multi[S, K, V] {
	m := &Multi[S, K, V]{
		population: newPopulation(),
		forward:    make(map[V]holders[K]),
		reverse:    make(map[K][]V),
	}
	m.overlay.create, m.overlay.flush = newDelta[K, V], m.flush
	return m
}

// Populate starts the bulk load and locks the index until it finishes. insert adds a
// committed mapping. finish ends the load and must be called exactly once; a non-nil
// err makes the index invalid, and every later read returns err.
func (m *Multi[S, K, V]) Populate() (insert func(K, []V), finish func(err error)) {
	m.mu.Lock()
	return m.put, func(err error) {
		m.population.finish(err)
		m.mu.Unlock()
	}
}

// Wait blocks until the bulk load finishes or ctx is done. It returns the load's error
// or ctx.Err().
func (m *Multi[S, K, V]) Wait(ctx context.Context) error {
	return m.population.wait(ctx)
}

// Set maps key to values in committed state, replacing any prior values of key.
func (m *Multi[S, K, V]) Set(key K, values []V) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.put(key, values)
}

// Delete removes key from committed state.
func (m *Multi[S, K, V]) Delete(key K) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.remove(key)
}

// Stage maps key to values in the changes of scope. The change applies to committed
// state when scope commits.
func (m *Multi[S, K, V]) Stage(scope S, key K, values []V) {
	if isNone(scope) {
		m.Set(key, values)
		return
	}
	m.overlay.load(scope).set(key, slices.Clone(values))
}

// StageDelete removes key in the changes of scope. The removal applies to committed
// state when scope commits.
func (m *Multi[S, K, V]) StageDelete(scope S, key K) {
	if isNone(scope) {
		m.Delete(key)
		return
	}
	m.overlay.load(scope).delete(key)
}

// Get returns the keys that hold any of values, with the changes of scope applied. It
// blocks until the bulk load finishes. The caller owns the returned slice, which is
// never nil.
func (m *Multi[S, K, V]) Get(scope S, values ...V) ([]K, error) {
	if len(values) == 0 {
		return []K{}, nil
	}
	<-m.population.done
	m.mu.RLock()
	defer m.mu.RUnlock()
	if err := m.population.err; err != nil {
		return nil, err
	}
	committed := set.New[K]()
	for _, v := range values {
		m.forward[v].addTo(committed)
	}
	return mergeValues(&m.overlay, scope, committed, values), nil
}

// Committing returns the keys that a transaction in its commit or an announced write
// maps to any of values. Get misses them until that write returns. The write can fail,
// so a caller must check each key against storage. Call Committing before Get.
func (m *Multi[S, K, V]) Committing(values ...V) []K {
	return m.appendKeys(committingKeys(&m.overlay, values), values)
}

// Announce makes Committing return key for each of values until Withdraw of the
// returned handle.
func (m *Multi[S, K, V]) Announce(key K, values []V) uint64 {
	return m.announce(key, slices.Clone(values))
}

func (m *Multi[S, K, V]) put(key K, values []V) {
	m.remove(key)
	if len(values) == 0 {
		return
	}
	values = slices.Clone(values)
	m.reverse[key] = values
	for _, v := range values {
		addHolder(m.forward, v, key)
	}
}

func (m *Multi[S, K, V]) remove(key K) {
	prev, ok := m.reverse[key]
	if !ok {
		return
	}
	for _, v := range prev {
		removeHolder(m.forward, v, key)
	}
	delete(m.reverse, key)
}

func (m *Multi[S, K, V]) flush(d *delta[K, V]) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for k, s := range d.state {
		if s.deleted {
			m.remove(k)
			continue
		}
		m.put(k, s.values)
	}
}
