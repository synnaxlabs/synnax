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

// Lookup is an exact-match index from values of type V to the primary keys of type K
// that hold them. Lookup is safe for concurrent use.
type Lookup[S Scope, K, V comparable] struct {
	// mu guards forward, reverse, and population.err. Populate holds it for writing
	// until the load finishes.
	mu         sync.RWMutex
	population population
	forward    map[V]keyList[K]
	reverse    map[K]V
	overlay    overlay[S, *delta[K, V]]
	// distinct is true when each value has about one holder, so the load sizes forward
	// for one value per key.
	distinct bool
	// loaded holds the mappings of the bulk load until it finishes. A map that grows
	// rehashes what it holds, so the load builds the maps once at their final size.
	loaded []mapping[K, V]
}

// NewLookup returns an empty Lookup that waits for its bulk load.
func NewLookup[S Scope, K, V comparable]() *Lookup[S, K, V] {
	l := &Lookup[S, K, V]{
		population: newPopulation(),
		forward:    make(map[V]keyList[K]),
		reverse:    make(map[K]V),
	}
	l.overlay.create, l.overlay.flush = newDelta[K, V], l.flush
	return l
}

// NewDistinctLookup returns an empty Lookup for values that each have about one holder,
// such as the values of a unique index. It waits for its bulk load.
func NewDistinctLookup[S Scope, K, V comparable]() *Lookup[S, K, V] {
	l := NewLookup[S, K, V]()
	l.distinct = true
	return l
}

// Populate starts the bulk load and locks the index until it finishes. insert adds a
// committed mapping. finish ends the load and must be called exactly once; a non-nil
// err makes the index invalid, and every later read returns err.
func (l *Lookup[S, K, V]) Populate() (insert func(K, V), finish func(err error)) {
	l.mu.Lock()
	return l.insert, func(err error) {
		if err == nil && len(l.loaded) > 0 {
			l.load()
		}
		l.loaded = nil
		l.population.finish(err)
		l.mu.Unlock()
	}
}

func (l *Lookup[S, K, V]) insert(key K, value V) {
	l.loaded = append(l.loaded, mapping[K, V]{key: key, value: value})
}

// mapping is one key and its value in a bulk load.
type mapping[K, V comparable] struct {
	key   K
	value V
}

// load maps each key of the bulk load to its value. The index holds nothing before.
func (l *Lookup[S, K, V]) load() {
	l.reverse = make(map[K]V, len(l.loaded))
	if l.distinct {
		l.forward = make(map[V]keyList[K], len(l.loaded))
	}
	for _, m := range l.loaded {
		l.put(m.key, m.value)
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
	l.overlay.load(scope).setOne(key, value)
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
	committed := set.New[K]()
	for _, v := range values {
		l.forward[v].addTo(committed)
	}
	return mergeValues(&l.overlay, scope, committed, values), nil
}

// Value returns the value that key maps to, with the changes of scope applied, and
// false when key maps to none. It blocks until the bulk load finishes, and returns the
// load's error.
func (l *Lookup[S, K, V]) Value(scope S, key K) (V, bool, error) {
	var none V
	<-l.population.done
	if err := l.population.err; err != nil {
		return none, false, err
	}
	if d, ok := l.overlay.get(scope); ok {
		if s, staged := d.state[key]; staged {
			if s.deleted {
				return none, false, nil
			}
			return s.values[0], true, nil
		}
	}
	l.mu.RLock()
	defer l.mu.RUnlock()
	v, ok := l.reverse[key]
	return v, ok, nil
}

// Holder returns a key that maps to value, with the changes of scope applied, and false
// when no key does. A non-nil skip is never returned. It blocks until the bulk load
// finishes, and returns the load's error.
func (l *Lookup[S, K, V]) Holder(scope S, value V, skip *K) (K, bool, error) {
	var none K
	<-l.population.done
	if err := l.population.err; err != nil {
		return none, false, err
	}
	other := func(k K) bool { return skip == nil || k != *skip }
	d, staged := l.overlay.get(scope)
	if staged {
		h := d.forward[value]
		if k, ok := h.find(other); ok {
			return k, true, nil
		}
	}
	l.mu.RLock()
	defer l.mu.RUnlock()
	k, found := l.forward[value].find(func(k K) bool {
		if !other(k) {
			return false
		}
		// A staged key holds its staged value, which the delta search saw.
		if staged {
			if _, changed := d.state[k]; changed {
				return false
			}
		}
		return true
	})
	return k, found, nil
}

// Committing returns the keys that a transaction in its commit maps to any of values.
// Get misses them until the commit flushes. The commit can fail, so a caller must check
// each key against storage. Call Committing before Get: a scope leaves the overlay only
// after its flush, so the two reads together miss no key.
func (l *Lookup[S, K, V]) Committing(values ...V) []K {
	return committingKeys(&l.overlay, values)
}

func (l *Lookup[S, K, V]) put(key K, value V) {
	if prev, ok := l.reverse[key]; ok {
		if prev == value {
			return
		}
		l.removeFromForward(key, prev)
	}
	keys := l.forward[value]
	keys.add(key)
	l.forward[value] = keys
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
	if keys.remove(key) {
		delete(l.forward, value)
		return
	}
	l.forward[value] = keys
}

// keyList is the keys that hold one value in a Lookup. The first key sits inline, so a
// value with one holder, the common case, allocates nothing.
type keyList[K comparable] struct {
	// first is a key when held is true.
	first K
	held  bool
	rest  []K
}

func (l *keyList[K]) add(key K) {
	if !l.held {
		l.first, l.held = key, true
		return
	}
	l.rest = append(l.rest, key)
}

// remove removes key, and reports whether the list is then empty.
func (l *keyList[K]) remove(key K) bool {
	if l.held && l.first == key {
		if n := len(l.rest); n > 0 {
			l.first, l.rest = l.rest[n-1], l.rest[:n-1]
		} else {
			var none K
			l.first, l.held = none, false
		}
	} else if i := slices.Index(l.rest, key); i >= 0 {
		l.rest = slices.Delete(l.rest, i, i+1)
	}
	return !l.held
}

// find returns a key that f accepts, and false when f accepts none.
func (l keyList[K]) find(f func(K) bool) (K, bool) {
	if l.held && f(l.first) {
		return l.first, true
	}
	if i := slices.IndexFunc(l.rest, f); i >= 0 {
		return l.rest[i], true
	}
	var none K
	return none, false
}

func (l keyList[K]) addTo(out set.Set[K]) {
	if l.held {
		out.Add(l.first)
	}
	out.Add(l.rest...)
}

func (l *Lookup[S, K, V]) flush(d *delta[K, V]) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for k, s := range d.state {
		if s.deleted {
			l.remove(k)
			continue
		}
		l.put(k, s.values[0])
	}
}
