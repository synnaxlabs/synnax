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
	// mu guards forward, reverse, and population.err. Populate holds it for writing
	// until the load finishes.
	mu         sync.RWMutex
	population population
	forward    map[V]holders[K]
	reverse    map[K][]V
	overlay    overlay[S, *multiDelta[K, V]]
	multiAnnouncements[K, V]
}

// NewMulti returns an empty Multi that waits for its bulk load.
func NewMulti[S Scope, K, V comparable]() *Multi[S, K, V] {
	m := &Multi[S, K, V]{
		population: newPopulation(),
		forward:    make(map[V]holders[K]),
		reverse:    make(map[K][]V),
	}
	m.overlay.create, m.overlay.flush = newMultiDelta[K, V], m.flush
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
	m.overlay.load(scope).set(key, values)
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
	if d, ok := m.overlay.get(scope); ok {
		return d.merge(committed, values), nil
	}
	return committed.Slice(), nil
}

// Committing returns the keys that a transaction in its commit or an announced write
// maps to any of values. Get misses them until that write returns. The write can fail,
// so a caller must check each key against storage. Call Committing before Get.
func (m *Multi[S, K, V]) Committing(values ...V) []K {
	m.overlay.mu.RLock()
	var keys []K
	for scope, d := range m.overlay.deltas {
		if !scope.Committing() {
			continue
		}
		for _, v := range values {
			keys = d.forward[v].appendTo(keys)
		}
	}
	m.overlay.mu.RUnlock()
	return m.appendKeys(keys, values)
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

func (m *Multi[S, K, V]) flush(d *multiDelta[K, V]) {
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

// holdersSetSize is the number of holders past which holders switches from a slice to
// a set. Most values have few holders, and a slice costs far less memory than a map.
const holdersSetSize = 32

// holders is the set of keys that hold one value.
type holders[K comparable] struct {
	few  []K
	many set.Set[K]
}

func (h *holders[K]) add(key K) {
	if h.many != nil {
		h.many.Add(key)
		return
	}
	if slices.Contains(h.few, key) {
		return
	}
	if len(h.few) < holdersSetSize {
		h.few = append(h.few, key)
		return
	}
	h.many = set.New(append(h.few, key)...)
	h.few = nil
}

func (h *holders[K]) remove(key K) {
	if h.many != nil {
		h.many.Remove(key)
		return
	}
	if i := slices.Index(h.few, key); i >= 0 {
		h.few = slices.Delete(h.few, i, i+1)
	}
}

func (h holders[K]) len() int {
	if h.many != nil {
		return len(h.many)
	}
	return len(h.few)
}

func (h holders[K]) addTo(out set.Set[K]) {
	if h.many != nil {
		for k := range h.many {
			out.Add(k)
		}
		return
	}
	out.Add(h.few...)
}

func (h holders[K]) appendTo(keys []K) []K {
	if h.many != nil {
		return slices.AppendSeq(keys, maps.Keys(h.many))
	}
	return append(keys, h.few...)
}

func addHolder[K, V comparable](forward map[V]holders[K], value V, key K) {
	h := forward[value]
	h.add(key)
	forward[value] = h
}

func removeHolder[K, V comparable](forward map[V]holders[K], value V, key K) {
	h, ok := forward[value]
	if !ok {
		return
	}
	h.remove(key)
	if h.len() == 0 {
		delete(forward, value)
		return
	}
	forward[value] = h
}

// multiStaged is the state of one key in a multiDelta.
type multiStaged[V comparable] struct {
	// values is the staged set of values. Nil when deleted is true.
	values  []V
	deleted bool
}

// multiDelta is the set of changes that one scope has staged against a Multi.
type multiDelta[K, V comparable] struct {
	// state is the staged state of each key. A key in state overrides committed state.
	state map[K]multiStaged[V]
	// forward maps each staged value to the keys that hold it.
	forward map[V]holders[K]
}

func newMultiDelta[K, V comparable]() *multiDelta[K, V] {
	return &multiDelta[K, V]{
		state:   make(map[K]multiStaged[V]),
		forward: make(map[V]holders[K]),
	}
}

func (d *multiDelta[K, V]) isEmpty() bool { return len(d.state) == 0 }

func (d *multiDelta[K, V]) set(key K, values []V) {
	d.unstage(key)
	values = slices.Clone(values)
	d.state[key] = multiStaged[V]{values: values}
	for _, v := range values {
		addHolder(d.forward, v, key)
	}
}

func (d *multiDelta[K, V]) delete(key K) {
	d.unstage(key)
	d.state[key] = multiStaged[V]{deleted: true}
}

func (d *multiDelta[K, V]) unstage(key K) {
	prev, ok := d.state[key]
	if !ok || prev.deleted {
		return
	}
	for _, v := range prev.values {
		removeHolder(d.forward, v, key)
	}
}

// merge returns the keys holding any of values once the delta applies to committed.
func (d *multiDelta[K, V]) merge(committed set.Set[K], values []V) []K {
	for k, s := range d.state {
		if s.deleted || !slices.ContainsFunc(s.values, func(v V) bool {
			return slices.Contains(values, v)
		}) {
			committed.Remove(k)
		}
	}
	for _, v := range values {
		d.forward[v].addTo(committed)
	}
	return committed.Slice()
}

// multiAnnouncement is the set of values that key takes when a write in progress
// finishes.
type multiAnnouncement[K, V comparable] struct {
	handle uint64
	key    K
	values []V
}

// multiAnnouncements holds the writes in progress outside any transaction. Its zero
// value is ready to use.
type multiAnnouncements[K, V comparable] struct {
	mu      sync.RWMutex
	entries []multiAnnouncement[K, V]
	// last is the handle of the latest Announce.
	last uint64
}

// Announce makes Committing return key for each of values until Withdraw of the
// returned handle.
func (a *multiAnnouncements[K, V]) Announce(key K, values []V) uint64 {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.last++
	a.entries = append(a.entries, multiAnnouncement[K, V]{
		handle: a.last,
		key:    key,
		values: slices.Clone(values),
	})
	return a.last
}

// Withdraw ends the Announce that returned handle. Apply a finished write to committed
// state first, so that a Committing call followed by Get misses no key.
func (a *multiAnnouncements[K, V]) Withdraw(handle uint64) {
	a.mu.Lock()
	defer a.mu.Unlock()
	i := slices.IndexFunc(a.entries, func(e multiAnnouncement[K, V]) bool {
		return e.handle == handle
	})
	a.entries = slices.Delete(a.entries, i, i+1)
}

func (a *multiAnnouncements[K, V]) appendKeys(keys []K, values []V) []K {
	a.mu.RLock()
	defer a.mu.RUnlock()
	for _, e := range a.entries {
		if slices.ContainsFunc(e.values, func(v V) bool {
			return slices.Contains(values, v)
		}) {
			keys = append(keys, e.key)
		}
	}
	return keys
}
