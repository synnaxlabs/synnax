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
	"maps"
	"slices"
	"sync"

	"github.com/synnaxlabs/x/set"
)

// Multi is an exact-match index from values of type V to the primary keys of type K
// that hold them. Unlike Lookup, one key can hold any number of values. Multi is safe
// for concurrent use.
type Multi[S Scope, K, V comparable] struct {
	// mu guards forward, reverse, none, and population.err. Populate holds it for
	// writing until the load finishes.
	mu         sync.RWMutex
	population population
	forward    map[V]holders[K]
	reverse    map[K][]V
	// none holds the keys that hold no value.
	none    set.Set[K]
	overlay overlay[S, *delta[K, V]]
}

// NewMulti returns an empty Multi that waits for its bulk load.
func NewMulti[S Scope, K, V comparable]() *Multi[S, K, V] {
	m := &Multi[S, K, V]{
		population: newPopulation(),
		forward:    make(map[V]holders[K]),
		reverse:    make(map[K][]V),
		none:       make(set.Set[K]),
	}
	m.overlay.init(newDelta[K, V], m.flush)
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
	d := m.overlay.load(scope)
	// A gone key is in none, so staging no values for it again changes nothing.
	if d.gone.Contains(key) {
		delete(d.gone, key)
		if len(values) == 0 {
			return
		}
	}
	if len(values) == 0 && m.stageFresh(d, key) {
		return
	}
	delete(d.fresh, key)
	d.set(key, slices.Clone(values))
}

// stageFresh adds key to the fresh keys of d when key holds nothing in committed state
// or in d. It returns false when key holds something, or before the bulk load finishes.
func (m *Multi[S, K, V]) stageFresh(d *delta[K, V], key K) bool {
	if _, staged := d.state[key]; staged || !m.absent(key) {
		return false
	}
	if d.fresh == nil {
		d.fresh = make(set.Set[K])
	}
	d.fresh.Add(key)
	d.grew(len(d.fresh))
	return true
}

// stageGone adds key to the gone keys of d when key is in none and d does not stage it.
// It returns false otherwise, or before the bulk load finishes.
func (m *Multi[S, K, V]) stageGone(d *delta[K, V], key K) bool {
	if _, staged := d.state[key]; staged || !m.inNone(key) {
		return false
	}
	if d.gone == nil {
		d.gone = make(set.Set[K])
	}
	d.gone.Add(key)
	d.grew(len(d.gone))
	return true
}

// absent reports whether committed state has no entry for key, in reverse or in none.
// It is false before the bulk load finishes, and after the load fails.
func (m *Multi[S, K, V]) absent(key K) bool {
	if !m.loaded() {
		return false
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, held := m.reverse[key]
	return m.population.err == nil && !held && !m.none.Contains(key)
}

// inNone reports whether key is in none in committed state. It is false before the bulk
// load finishes, and after the load fails.
func (m *Multi[S, K, V]) inNone(key K) bool {
	if !m.loaded() {
		return false
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.population.err == nil && m.none.Contains(key)
}

// loaded reports whether the bulk load has finished.
func (m *Multi[S, K, V]) loaded() bool {
	select {
	case <-m.population.done:
		return true
	default:
		return false
	}
}

// Current reports whether key holds exactly values, with the changes of scope applied,
// so that staging values for key would change nothing. It is false until the bulk load
// finishes, and false after the load fails.
func (m *Multi[S, K, V]) Current(scope S, key K, values []V) bool {
	select {
	case <-m.population.done:
	default:
		return false
	}
	if d, ok := m.overlay.get(scope); ok {
		if s, ok := d.state[key]; ok {
			return !s.deleted && slices.Equal(s.values, values)
		}
		if d.gone.Contains(key) {
			return false
		}
		if d.fresh.Contains(key) {
			return len(values) == 0
		}
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.population.err != nil {
		return false
	}
	if prev, ok := m.reverse[key]; ok {
		return slices.Equal(prev, values)
	}
	return len(values) == 0 && m.none.Contains(key)
}

// StageDelete removes key in the changes of scope. The removal applies to committed
// state when scope commits.
func (m *Multi[S, K, V]) StageDelete(scope S, key K) {
	if isNone(scope) {
		m.Delete(key)
		return
	}
	d := m.overlay.load(scope)
	// A fresh key holds nothing in committed state, so its delete changes nothing.
	if d.fresh.Contains(key) {
		delete(d.fresh, key)
		return
	}
	if !m.stageGone(d, key) {
		d.delete(key)
	}
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

// GetEach returns, at each position of values, the keys that hold that value, as Get
// does for one value, under one wait and one lock. A position whose value no key holds
// is nil, and GetEach returns nil when no key holds any of values.
func (m *Multi[S, K, V]) GetEach(scope S, values []V) ([][]K, error) {
	<-m.population.done
	m.mu.RLock()
	defer m.mu.RUnlock()
	if err := m.population.err; err != nil {
		return nil, err
	}
	var out [][]K
	d, staged := m.overlay.get(scope)
	for i, v := range values {
		h, held := m.forward[v]
		if !staged && !held {
			continue
		}
		if out == nil {
			out = make([][]K, len(values))
		}
		if !staged {
			out[i] = h.appendTo(nil)
			continue
		}
		committed := set.New[K]()
		h.addTo(committed)
		out[i] = d.merge(committed, values[i:i+1])
	}
	return out, nil
}

// Empty reports whether no key holds a value in committed state, in the changes of
// scope, or in the changes of a scope in its commit. It is false until the bulk load
// finishes, and after the load fails.
func (m *Multi[S, K, V]) Empty(scope S) bool {
	if !m.loaded() {
		return false
	}
	// The changes go first: a commit flushes them into committed state before it
	// empties them, so a holder in either is seen.
	o := &m.overlay
	o.mu.RLock()
	// Only the staging scope, or a scope in its commit, reads the changes.
	staged := !isNone(o.scope) && (o.scope == scope || o.scope.Committing()) &&
		len(o.d.forward) > 0
	o.mu.RUnlock()
	if staged {
		return false
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.population.err == nil && len(m.forward) == 0
}

// CommittingEach returns, at each position of values, the keys that a transaction in
// its commit maps to that value. See Lookup.Committing for its contract.
func (m *Multi[S, K, V]) CommittingEach(values []V) [][]K {
	return committingEach(&m.overlay, values)
}

// Committing returns the keys that a transaction in its commit maps to any of values.
// See Lookup.Committing for its contract.
func (m *Multi[S, K, V]) Committing(values ...V) []K {
	return committingKeys(&m.overlay, values)
}

// Holding returns the keys that hold at least one value when holding is true, and the
// keys that hold none when holding is false, with the changes of scope applied. It
// blocks until the bulk load finishes. The caller owns the returned slice, which is
// never nil.
func (m *Multi[S, K, V]) Holding(scope S, holding bool) ([]K, error) {
	<-m.population.done
	m.mu.RLock()
	defer m.mu.RUnlock()
	if err := m.population.err; err != nil {
		return nil, err
	}
	keys := make(set.Set[K])
	if holding {
		for k := range m.reverse {
			keys.Add(k)
		}
	} else {
		maps.Copy(keys, m.none)
	}
	if d, ok := m.overlay.get(scope); ok {
		d.applyHolding(keys, holding)
	}
	return keys.Slice(), nil
}

// CommittingHolding returns the keys that a transaction in its commit leaves holding
// at least one value when holding is true, or holding none when holding is false. See
// Lookup.Committing for its contract.
func (m *Multi[S, K, V]) CommittingHolding(holding bool) []K {
	keys := make(set.Set[K])
	m.overlay.mu.RLock()
	if d, ok := m.overlay.committing(); ok {
		d.applyHolding(keys, holding)
	}
	m.overlay.mu.RUnlock()
	return keys.Slice()
}

// HeldValues returns those of values that at least one key that in accepts holds, with
// the changes of scope applied. It blocks until the bulk load finishes, and calls in
// with the index locked.
func (m *Multi[S, K, V]) HeldValues(
	scope S,
	values []V,
	in func(K) bool,
) (set.Set[V], error) {
	<-m.population.done
	m.mu.RLock()
	defer m.mu.RUnlock()
	if err := m.population.err; err != nil {
		return nil, err
	}
	d, staged := m.overlay.get(scope)
	// kept reports whether a committed holder keeps its values under the changes.
	kept := func(k K) bool {
		if staged {
			if _, changed := d.state[k]; changed {
				return false
			}
		}
		return in(k)
	}
	held := make(set.Set[V])
	for _, v := range values {
		if m.forward[v].some(kept) || (staged && d.forward[v].some(in)) {
			held.Add(v)
		}
	}
	return held, nil
}

// Held returns the values that key holds, with the changes of scope applied. It
// returns false when the bulk load has not finished or failed.
func (m *Multi[S, K, V]) Held(scope S, key K) ([]V, bool) {
	select {
	case <-m.population.done:
	default:
		return nil, false
	}
	if d, ok := m.overlay.get(scope); ok {
		if s, ok := d.state[key]; ok {
			return s.values, true
		}
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.population.err != nil {
		return nil, false
	}
	return m.reverse[key], true
}

func (m *Multi[S, K, V]) put(key K, values []V) {
	if prev, ok := m.reverse[key]; ok && slices.Equal(prev, values) {
		return
	}
	if len(values) == 0 && m.none.Contains(key) {
		return
	}
	m.remove(key)
	if len(values) == 0 {
		m.none.Add(key)
		return
	}
	values = slices.Clone(values)
	m.reverse[key] = values
	for _, v := range values {
		addHolder(m.forward, v, key)
	}
}

func (m *Multi[S, K, V]) remove(key K) {
	m.none.Remove(key)
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
	maps.Copy(m.none, d.fresh)
	for k := range d.gone {
		m.none.Remove(k)
	}
}
