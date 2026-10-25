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
)

// Lookup is an exact-match index from values of type V to the primary keys of type K
// that hold them. Lookup is safe for concurrent use.
type Lookup[S Scope, K, V comparable] struct {
	// mu guards forward, reverse, and population.err. Populate holds it for writing
	// until the load finishes.
	mu         sync.RWMutex
	population population
	forward    map[V][]K
	reverse    map[K]V
	overlay    overlay[S, *delta[K, V]]
}

// NewLookup returns an empty Lookup that waits for its bulk load.
func NewLookup[S Scope, K, V comparable]() *Lookup[S, K, V] {
	l := &Lookup[S, K, V]{
		population: newPopulation(),
		forward:    make(map[V][]K),
		reverse:    make(map[K]V),
	}
	l.overlay.create, l.overlay.flush = newDelta[K, V], l.flush
	return l
}

// Populate starts the bulk load and locks the index until it finishes. insert adds a
// committed mapping. finish ends the load and must be called exactly once; a non-nil
// err makes the index invalid, and every later read returns err.
func (l *Lookup[S, K, V]) Populate() (insert func(K, V), finish func(err error)) {
	l.mu.Lock()
	return l.put, func(err error) {
		l.population.finish(err)
		l.mu.Unlock()
	}
}

// Wait blocks until the bulk load finishes or ctx is done. It returns the load's error
// or ctx.Err().
func (l *Lookup[S, K, V]) Wait(ctx context.Context) error {
	return l.population.wait(ctx)
}

// Set maps key to value in committed state, replacing any prior value of key.
func (l *Lookup[S, K, V]) Set(key K, value V) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.put(key, value)
}

// Delete removes key from committed state.
func (l *Lookup[S, K, V]) Delete(key K) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.remove(key)
}

// Stage maps key to value in the changes of scope. The change applies to committed
// state when scope commits.
func (l *Lookup[S, K, V]) Stage(scope S, key K, value V) {
	if isNone(scope) {
		l.Set(key, value)
		return
	}
	l.overlay.load(scope).set(key, value)
}

// StageDelete removes key in the changes of scope. The removal applies to committed
// state when scope commits.
func (l *Lookup[S, K, V]) StageDelete(scope S, key K) {
	if isNone(scope) {
		l.Delete(key)
		return
	}
	l.overlay.load(scope).delete(key)
}

// Get returns the keys mapped to any of values, with the changes of scope applied. It
// blocks until the bulk load finishes. The caller owns the returned slice, which is
// never nil.
func (l *Lookup[S, K, V]) Get(scope S, values ...V) ([]K, error) {
	if len(values) == 0 {
		return []K{}, nil
	}
	<-l.population.done
	l.mu.RLock()
	defer l.mu.RUnlock()
	if err := l.population.err; err != nil {
		return nil, err
	}
	var committed []K
	for _, v := range values {
		committed = append(committed, l.forward[v]...)
	}
	return mergeValues(&l.overlay, scope, committed, values), nil
}

func (l *Lookup[S, K, V]) put(key K, value V) {
	if prev, ok := l.reverse[key]; ok {
		if prev == value {
			return
		}
		l.removeFromForward(key, prev)
	}
	l.forward[value] = append(l.forward[value], key)
	l.reverse[key] = value
}

func (l *Lookup[S, K, V]) remove(key K) {
	prev, ok := l.reverse[key]
	if !ok {
		return
	}
	l.removeFromForward(key, prev)
	delete(l.reverse, key)
}

func (l *Lookup[S, K, V]) removeFromForward(key K, value V) {
	keys := l.forward[value]
	if i := slices.Index(keys, key); i >= 0 {
		keys = slices.Delete(keys, i, i+1)
	}
	if len(keys) == 0 {
		delete(l.forward, value)
		return
	}
	l.forward[value] = keys
}

func (l *Lookup[S, K, V]) flush(d *delta[K, V]) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for k, s := range d.state {
		if s.deleted {
			l.remove(k)
			continue
		}
		l.put(k, s.value)
	}
}
