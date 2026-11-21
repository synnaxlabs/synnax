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
	"maps"
	"slices"
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
	// Committing reports whether the transaction is in its commit. Its staged changes
	// are final from that point.
	Committing() bool
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

// wait returns the load's error once the load finishes, or ctx.Err() if ctx ends first.
// A finished load wins over an ended ctx.
func (p *population) wait(ctx context.Context) error {
	select {
	case <-p.done:
		return p.err
	default:
	}
	select {
	case <-p.done:
		return p.err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// staged is the state of one key in a delta.
type staged[V comparable] struct {
	// values is the staged set of values. Nil when deleted is true.
	values  []V
	deleted bool
}

// delta is the set of changes that one scope has staged against an index.
type delta[K, V comparable] struct {
	// state is the staged state of each key. A key in state overrides committed state.
	state map[K]staged[V]
	// forward maps each staged value to the keys that hold it. It is nil until a staged
	// key holds a value.
	forward map[V]holders[K]
	// arena backs the values of setOne, and keys the first holder of each staged value,
	// so a batch of staged keys allocates a few arrays instead of slices per key.
	arena []V
	keys  []K
	// fresh holds the keys staged with no value that held nothing before, and gone the
	// keys in none that the scope deleted. Neither needs an entry in state. Only Multi
	// stages them.
	fresh, gone set.Set[K]
	// peak is the most entries that one map of d held since the last reset. A cleared
	// map keeps the room of its peak, so reset drops the maps by it.
	peak int
}

func newDelta[K, V comparable]() *delta[K, V] {
	return &delta[K, V]{state: make(map[K]staged[V])}
}

func (d *delta[K, V]) isEmpty() bool {
	return len(d.state) == 0 && len(d.fresh) == 0 && len(d.gone) == 0
}

// reset empties d for the next scope. It drops the maps when one grew past retained, so
// that one large transaction does not hold their memory.
func (d *delta[K, V]) reset() {
	if d.peak > retained {
		d.state, d.forward, d.fresh, d.gone = make(map[K]staged[V]), nil, nil, nil
	} else {
		clear(d.state)
		clear(d.forward)
		clear(d.fresh)
		clear(d.gone)
	}
	d.arena = truncated(d.arena)
	d.keys = truncated(d.keys)
	d.peak = 0
}

// retained is the most entries that a map of a delta keeps room for between scopes.
const retained = 1024

// truncated returns s with no elements, or nil when s grew past retained.
func truncated[T any](s []T) []T {
	if cap(s) > retained {
		return nil
	}
	clear(s)
	return s[:0]
}

// set stages values for key. The delta owns values from then on.
func (d *delta[K, V]) set(key K, values []V) {
	d.unstage(key)
	d.state[key] = staged[V]{values: values}
	if len(values) > 0 && d.forward == nil {
		d.forward = make(map[V]holders[K])
	}
	for _, v := range values {
		d.addHolder(v, key)
	}
	d.grew(max(len(d.state), len(d.forward)))
}

// setOne stages value as the one value of key.
func (d *delta[K, V]) setOne(key K, value V) {
	if len(d.arena) == cap(d.arena) {
		d.arena = make([]V, 0, max(arenaMin, 2*cap(d.arena)))
	}
	d.arena = append(d.arena, value)
	n := len(d.arena)
	d.set(key, d.arena[n-1:n:n])
}

// addHolder adds key to the staged holders of value. A first holder takes its slice
// from keys.
func (d *delta[K, V]) addHolder(value V, key K) {
	h := d.forward[value]
	if h.len() > 0 {
		h.add(key)
		d.forward[value] = h
		return
	}
	if len(d.keys) == cap(d.keys) {
		d.keys = make([]K, 0, max(arenaMin, 2*cap(d.keys)))
	}
	d.keys = append(d.keys, key)
	n := len(d.keys)
	h.few = d.keys[n-1 : n : n]
	d.forward[value] = h
}

// arenaMin is the size of the first arena of a delta.
const arenaMin = 8

// grew records that a map of d holds n entries.
func (d *delta[K, V]) grew(n int) { d.peak = max(d.peak, n) }

func (d *delta[K, V]) delete(key K) {
	d.unstage(key)
	d.state[key] = staged[V]{deleted: true}
	d.grew(len(d.state))
}

func (d *delta[K, V]) unstage(key K) {
	prev, ok := d.state[key]
	if !ok || prev.deleted {
		return
	}
	for _, v := range prev.values {
		removeHolder(d.forward, v, key)
	}
}

// merge returns the keys holding any of values once the delta applies to committed.
func (d *delta[K, V]) merge(committed set.Set[K], values []V) []K {
	in := AnyOf(values)
	for k, s := range d.state {
		if s.deleted || !slices.ContainsFunc(s.values, in) {
			committed.Remove(k)
		}
	}
	for _, v := range values {
		d.forward[v].addTo(committed)
	}
	return committed.Slice()
}

// applyHolding adds to keys each key that the delta leaves holding at least one value
// when holding is true, or holding none when holding is false, and removes each other
// key the delta changes.
func (d *delta[K, V]) applyHolding(keys set.Set[K], holding bool) {
	for k, s := range d.state {
		if !s.deleted && (len(s.values) > 0) == holding {
			keys.Add(k)
		} else {
			keys.Remove(k)
		}
	}
	if !holding {
		maps.Copy(keys, d.fresh)
	}
	for k := range d.gone {
		keys.Remove(k)
	}
}

// anyOfSetSize is the number of values past which AnyOf builds a set. A scan of a few
// values costs less than a hash.
const anyOfSetSize = 8

// AnyOf returns a predicate that reports whether a value is any of values. Build it
// once and call it for each candidate. The caller must not change values while the
// predicate is in use.
func AnyOf[V comparable](values []V) func(V) bool {
	if len(values) <= anyOfSetSize {
		return func(v V) bool { return slices.Contains(values, v) }
	}
	return set.New(values...).Contains
}

// changes is the set of changes that one scope has staged against an index.
type changes interface {
	isEmpty() bool
	reset()
}

// overlay holds the changes that the writing scope has staged against an index. The DB
// lets one transaction write at a time, and ends it before the next writes, so one set
// of changes serves every scope.
type overlay[S Scope, D changes] struct {
	// mu guards scope, and d against the reads of committing.
	mu sync.RWMutex
	// scope is the scope that staged d, or the zero S when no scope has staged since
	// the last one ended.
	scope S
	// d is made on the first load, so an index that no scope stages to costs nothing.
	d    D
	made bool
	// create returns empty changes.
	create func() D
	// flush applies the changes of a committed scope to committed state.
	flush func(D)
	// end is finish, bound once so that registering it allocates nothing.
	end func(committed bool)
}

// init gives o create to make its changes, and flush to apply them.
func (o *overlay[S, D]) init(create func() D, flush func(D)) {
	o.create, o.flush = create, flush
}

// load returns the changes of scope. The first load of a scope registers a cleanup on
// scope that flushes the changes if scope committed, and then empties them. It panics
// when another scope staged changes and has not ended.
func (o *overlay[S, D]) load(scope S) D {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.scope != scope {
		if !isNone(o.scope) {
			panic("[index] - a scope staged changes while another was open")
		}
		if !o.made {
			o.d, o.end, o.made = o.create(), o.finish, true
		}
		o.scope = scope
		scope.OnCleanup(o.end)
	}
	return o.d
}

// finish flushes the changes when their scope committed, and then empties them.
func (o *overlay[S, D]) finish(committed bool) {
	if committed && !o.d.isEmpty() {
		o.flush(o.d)
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	o.d.reset()
	var none S
	o.scope = none
}

// get returns the changes of scope, or false when scope has staged none.
func (o *overlay[S, D]) get(scope S) (D, bool) {
	var none D
	if isNone(scope) {
		return none, false
	}
	o.mu.RLock()
	staged := o.scope == scope && !o.d.isEmpty()
	o.mu.RUnlock()
	if !staged {
		return none, false
	}
	return o.d, true
}

// committing returns the changes of the scope in its commit, or false when no scope is
// in its commit. Call it with mu held for reading, and read the changes before
// releasing it.
func (o *overlay[S, D]) committing() (D, bool) {
	if isNone(o.scope) || !o.scope.Committing() {
		var none D
		return none, false
	}
	return o.d, true
}

// mergeValues returns committed with the delta of scope applied for values. The result
// is never nil.
func mergeValues[S Scope, K, V comparable](
	o *overlay[S, *delta[K, V]],
	scope S,
	committed set.Set[K],
	values []V,
) []K {
	if d, ok := o.get(scope); ok {
		return d.merge(committed, values)
	}
	return committed.Slice()
}

// committingKeys returns the keys that committing scopes map to any of values.
func committingKeys[S Scope, K, V comparable](
	o *overlay[S, *delta[K, V]],
	values []V,
) []K {
	o.mu.RLock()
	defer o.mu.RUnlock()
	d, ok := o.committing()
	if !ok {
		return nil
	}
	var keys []K
	for _, v := range values {
		keys = d.forward[v].appendTo(keys)
	}
	return keys
}

// committingEach returns, at each position of values, the keys that committing scopes
// map to that value. It returns nil when no scope is committing.
func committingEach[S Scope, K, V comparable](
	o *overlay[S, *delta[K, V]],
	values []V,
) [][]K {
	o.mu.RLock()
	defer o.mu.RUnlock()
	d, ok := o.committing()
	if !ok {
		return nil
	}
	out := make([][]K, len(values))
	for i, v := range values {
		out[i] = d.forward[v].appendTo(out[i])
	}
	return out
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

// find returns a key of h that f accepts, and false when f accepts none.
func (h holders[K]) find(f func(K) bool) (K, bool) {
	if h.many == nil {
		if i := slices.IndexFunc(h.few, f); i >= 0 {
			return h.few[i], true
		}
	}
	for k := range h.many {
		if f(k) {
			return k, true
		}
	}
	var none K
	return none, false
}

// some reports whether f accepts any key of h.
func (h holders[K]) some(f func(K) bool) bool {
	if h.many == nil {
		return slices.ContainsFunc(h.few, f)
	}
	for k := range h.many {
		if f(k) {
			return true
		}
	}
	return false
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
