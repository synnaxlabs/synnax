// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package index implements in-memory secondary indexes that are loaded once in bulk,
// then kept current by single writes, with changes staged per transaction.
//
// Every read blocks until the bulk load finishes. A load that fails makes the index
// invalid for good: every later read returns the load's error.
package index

import (
	"context"
	"sync"

	"github.com/synnaxlabs/x/set"
)

// Scope is an open transaction that index changes can be staged against. The zero value
// of a Scope type stands for no transaction: changes staged against it apply to
// committed state at once, and reads against it see committed state only.
type Scope interface {
	comparable
	// OnCleanup registers fn to run once when the transaction ends. committed is true
	// when the transaction committed.
	OnCleanup(fn func(committed bool))
}

func isNone[S Scope](scope S) bool {
	var none S
	return scope == none
}

// population is the state of an index's one bulk load.
type population struct {
	// done closes when the load finishes.
	done chan struct{}
	// err is the error the load finished with. It is written under the index's write
	// lock before done closes.
	err error
}

func newPopulation() population { return population{done: make(chan struct{})} }

func (p *population) finish(err error) {
	p.err = err
	close(p.done)
}

func (p *population) wait(ctx context.Context) error {
	select {
	case <-p.done:
		return p.err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// staged is the state of one key in a delta.
type staged[V comparable] struct {
	// value is the staged value. Zero when deleted is true.
	value   V
	deleted bool
}

// delta is the set of changes that one scope has staged against an index.
type delta[K, V comparable] struct {
	// state is the staged state of each key. A key in state overrides committed state.
	state map[K]staged[V]
	// forward maps each staged value to the keys that hold it.
	forward map[V]set.Set[K]
}

func newDelta[K, V comparable]() *delta[K, V] {
	return &delta[K, V]{
		state:   make(map[K]staged[V]),
		forward: make(map[V]set.Set[K]),
	}
}

func (d *delta[K, V]) isEmpty() bool { return len(d.state) == 0 }

func (d *delta[K, V]) set(key K, value V) {
	if prev, ok := d.state[key]; ok && !prev.deleted {
		d.removeFromForward(key, prev.value)
	}
	d.state[key] = staged[V]{value: value}
	bucket, ok := d.forward[value]
	if !ok {
		bucket = set.New[K]()
		d.forward[value] = bucket
	}
	bucket.Add(key)
}

func (d *delta[K, V]) delete(key K) {
	if prev, ok := d.state[key]; ok && !prev.deleted {
		d.removeFromForward(key, prev.value)
	}
	d.state[key] = staged[V]{deleted: true}
}

func (d *delta[K, V]) removeFromForward(key K, value V) {
	bucket, ok := d.forward[value]
	if !ok {
		return
	}
	delete(bucket, key)
	if len(bucket) == 0 {
		delete(d.forward, value)
	}
}

// merge returns the keys mapped to any of values once the delta applies to committed.
func (d *delta[K, V]) merge(committed []K, values []V) []K {
	result := set.New(committed...)
	valueSet := set.New(values...)
	for k, s := range d.state {
		if s.deleted || !valueSet.Contains(s.value) {
			result.Remove(k)
		}
	}
	for _, v := range values {
		for k := range d.forward[v] {
			result.Add(k)
		}
	}
	return result.Slice()
}

// changes is the set of changes that one scope has staged against an index.
type changes interface{ isEmpty() bool }

// overlay holds the changes of every scope that has staged changes against an index.
type overlay[S Scope, D changes] struct {
	// mu guards deltas.
	mu     sync.Mutex
	deltas map[S]D
	// create returns an empty set of changes.
	create func() D
	// flush applies the changes of a committed scope to committed state.
	flush func(D)
}

// load returns the changes of scope, creating them on first use. New changes register
// a cleanup on scope that drops them and flushes them if scope committed.
func (o *overlay[S, D]) load(scope S) D {
	o.mu.Lock()
	defer o.mu.Unlock()
	if d, ok := o.deltas[scope]; ok {
		return d
	}
	if o.deltas == nil {
		o.deltas = make(map[S]D)
	}
	d := o.create()
	o.deltas[scope] = d
	scope.OnCleanup(func(committed bool) {
		o.mu.Lock()
		delete(o.deltas, scope)
		o.mu.Unlock()
		if committed && !d.isEmpty() {
			o.flush(d)
		}
	})
	return d
}

// get returns the changes of scope, or false when scope has staged none.
func (o *overlay[S, D]) get(scope S) (D, bool) {
	var none D
	if isNone(scope) {
		return none, false
	}
	o.mu.Lock()
	d, ok := o.deltas[scope]
	o.mu.Unlock()
	if !ok || d.isEmpty() {
		return none, false
	}
	return d, true
}

// mergeValues returns committed with the delta of scope applied for values. The result
// is never nil.
func mergeValues[S Scope, K, V comparable](
	o *overlay[S, *delta[K, V]],
	scope S,
	committed []K,
	values []V,
) []K {
	if committed == nil {
		committed = []K{}
	}
	if d, ok := o.get(scope); ok {
		return d.merge(committed, values)
	}
	return committed
}
