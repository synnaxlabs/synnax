// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package gorp

import (
	"context"
	"fmt"
	"slices"
	"sync/atomic"

	"github.com/synnaxlabs/gorp/internal/index"
	"github.com/synnaxlabs/gorp/internal/key"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
)

// UniqueError is the error of a write whose value another entry of the same space
// holds. It wraps query.ErrUniqueViolation.
type UniqueError struct {
	// Space is the space of the unique index.
	Space string
	// Holder is the entry that holds the value.
	Holder Target
	// Value is the value that clashed.
	Value any
}

// Error implements error.
func (e UniqueError) Error() string {
	return fmt.Sprintf(
		"[gorp] - %s %s already holds %v in %s",
		e.Holder.Table,
		e.Holder.Key,
		e.Value,
		e.Space,
	)
}

// Unwrap returns query.ErrUniqueViolation.
func (e UniqueError) Unwrap() error { return query.ErrUniqueViolation }

// UniqueIndex is an in-memory index that refuses a write of a value that another entry
// of its space holds. Every UniqueIndex of one space on a DB shares one set of values,
// across tables. Construct via NewUniqueIndex and register on a Table through
// TableConfig.Indexes.
//
// Every write of an entry through the Table checks its value: a Create, an Update, and
// the detach of a delete. A write that keeps the value its entry holds passes, so an
// entry with a duplicate value from before the index stays writable. A Writer outside
// the Table and NewUncheckedWriter check nothing. The check waits for every index of
// the space to populate, and fails when one failed to.
type UniqueIndex[K Key, E Entry[K], V comparable] struct {
	name  string
	space string
	// value returns the value of an entry, and false when the entry holds none.
	value func(e E) (V, bool)
	store *index.Lookup[*txState, K, V]
	// members is the space of the index on the DB of its table, registry is the
	// registry of that DB, and table is the name of that table. All are set when the
	// index joins its space.
	members  *uniqueSpace[V]
	registry *registry
	table    string
}

// NewUniqueIndex constructs a UniqueIndex named name in the space named space. value
// returns the value of an entry, or false to exempt the entry. Every index of a space
// must have the same value type V, and a table can have at most one index in a space.
func NewUniqueIndex[K Key, E Entry[K], V comparable](
	name string,
	space string,
	value func(e E) (V, bool),
) *UniqueIndex[K, E, V] {
	return &UniqueIndex[K, E, V]{
		name:  name,
		space: space,
		value: value,
		store: index.NewLookup[*txState, K, V](),
	}
}

// Name implements Index.
func (u *UniqueIndex[K, E, V]) Name() string { return u.name }

func (u *UniqueIndex[K, E, V]) populate() (func(E), func(error)) {
	insert, finish := u.store.Populate()
	return func(entry E) {
			if v, ok := u.value(entry); ok {
				insert(entry.GorpKey(), v)
			}
		},
		func(err error) { finish(populateErr(u.name, err)) }
}

func (u *UniqueIndex[K, E, V]) set(entry E) {
	if v, ok := u.value(entry); ok {
		u.store.Set(entry.GorpKey(), v)
		return
	}
	u.store.Delete(entry.GorpKey())
}

func (u *UniqueIndex[K, E, V]) delete(key K) { u.store.Delete(key) }

func (u *UniqueIndex[K, E, V]) stageSet(tx Tx, entry E) {
	if v, ok := u.value(entry); ok {
		u.store.Stage(tx.txIdentity(), entry.GorpKey(), v)
		return
	}
	u.store.StageDelete(tx.txIdentity(), entry.GorpKey())
}

func (u *UniqueIndex[K, E, V]) stageDelete(tx Tx, key K) {
	u.store.StageDelete(tx.txIdentity(), key)
}

// Check returns the UniqueError that a write of e through the table of u would return:
// when another entry of the space holds the value of e in tx, unless the entry of e
// already holds that value in tx. Each write of the table runs it, so call it only to
// fail before work that a refused write would waste.
func (u *UniqueIndex[K, E, V]) Check(ctx context.Context, tx Tx, e *E) error {
	v, ok := u.value(*e)
	if !ok {
		return nil
	}
	if err := u.store.Wait(ctx); err != nil {
		return err
	}
	scope := tx.txIdentity()
	k := (*e).GorpKey()
	held, holds, err := u.store.Value(scope, k)
	if err != nil || (holds && held == v) {
		return err
	}
	if other, found, err := u.store.Holder(scope, v, &k); err != nil || found {
		return u.clash(NewTarget(u.table, other), v, err)
	}
	for _, m := range *u.members.members.Load() {
		if m == uniqueMember[V](u) {
			continue
		}
		if holder, found, err := m.holder(ctx, scope, v); err != nil || found {
			return u.clash(holder, v, err)
		}
	}
	return nil
}

// clash returns err, or else the UniqueError for holder and v.
func (u *UniqueIndex[K, E, V]) clash(holder Target, v V, err error) error {
	if err != nil {
		return err
	}
	return UniqueError{Space: u.space, Holder: holder, Value: v}
}

// release stages, in tx, the removal of the values of the entries with the text keys
// keys.
func (u *UniqueIndex[K, E, V]) release(tx Tx, keys []string) error {
	for _, text := range keys {
		k, err := key.Parse[K](text)
		if err != nil {
			return err
		}
		u.store.StageDelete(tx.txIdentity(), k)
	}
	return nil
}

// holder implements uniqueMember.
func (u *UniqueIndex[K, E, V]) holder(
	ctx context.Context,
	scope *txState,
	v V,
) (Target, bool, error) {
	if err := u.store.Wait(ctx); err != nil {
		return Target{}, false, err
	}
	k, found, err := u.store.Holder(scope, v, nil)
	if err != nil || !found {
		return Target{}, false, err
	}
	return NewTarget(u.table, k), true, nil
}

// Value returns the value of the entry of the table of u under key, as seen by snap,
// and false when no entry has key.
func (u *UniqueIndex[K, E, V]) Value(
	ctx context.Context,
	snap Snapshot,
	key K,
) (V, bool, error) {
	var none V
	if err := u.store.Wait(ctx); err != nil {
		return none, false, err
	}
	return u.store.Value(scopeOf(snap), key)
}

// ValueOf returns the value of the entry target, in any table of the space of u, as
// seen by snap, and false when its table has no entry under its key. It fails when no
// index of the space is on the table of target.
func (u *UniqueIndex[K, E, V]) ValueOf(
	ctx context.Context,
	snap Snapshot,
	target Target,
) (V, bool, error) {
	for _, m := range *u.members.members.Load() {
		if m.on(target.Table) {
			return m.valueOf(ctx, scopeOf(snap), target)
		}
	}
	var none V
	return none, false, errors.Newf(
		"[gorp] - no index of space %s is on table %s",
		u.space,
		target.Table,
	)
}

// on implements uniqueMember.
func (u *UniqueIndex[K, E, V]) on(table string) bool { return u.table == table }

// valueOf implements uniqueMember.
func (u *UniqueIndex[K, E, V]) valueOf(
	ctx context.Context,
	scope *txState,
	target Target,
) (V, bool, error) {
	var none V
	k, err := key.Parse[K](target.Key)
	if err != nil {
		return none, false, err
	}
	if err = u.store.Wait(ctx); err != nil {
		return none, false, err
	}
	return u.store.Value(scope, k)
}

// spaceName implements checkedIndex.
func (u *UniqueIndex[K, E, V]) spaceName() string { return u.space }

// canJoin implements checkedIndex.
func (u *UniqueIndex[K, E, V]) canJoin(r *registry) error {
	if found, ok := r.spaces[u.space]; ok {
		if _, ok = found.(*uniqueSpace[V]); !ok {
			return errors.Newf(
				"[gorp] - space %q holds values of another type than %T",
				u.space,
				*new(V),
			)
		}
	}
	return nil
}

// join implements checkedIndex.
func (u *UniqueIndex[K, E, V]) join(r *registry, table string) {
	space, ok := r.spaces[u.space].(*uniqueSpace[V])
	if !ok {
		space = &uniqueSpace[V]{}
		space.members.Store(&[]uniqueMember[V]{})
		r.spaces[u.space] = space
	}
	joined := append(slices.Clone(*space.members.Load()), uniqueMember[V](u))
	space.members.Store(&joined)
	u.members, u.registry, u.table = space, r, table
}

// leave implements checkedIndex.
func (u *UniqueIndex[K, E, V]) leave() {
	r := u.registry
	if r == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	kept := slices.DeleteFunc(
		slices.Clone(*u.members.members.Load()),
		func(o uniqueMember[V]) bool { return o == uniqueMember[V](u) },
	)
	if len(kept) == 0 {
		delete(r.spaces, u.space)
	}
	u.members.members.Store(&kept)
	u.registry = nil
}

// checkedIndex is an index whose value a write of its Table checks.
type checkedIndex[K Key, E Entry[K]] interface {
	Index[K, E]
	// Check returns a UniqueError when another entry holds the value of e in tx.
	Check(ctx context.Context, tx Tx, e *E) error
	// release stages, in tx, the removal of the values of the entries with the text
	// keys keys.
	release(tx Tx, keys []string) error
	// spaceName returns the name of the space of the index.
	spaceName() string
	// canJoin reports why the index cannot join its space in r. Call it with the lock
	// of r held.
	canJoin(r *registry) error
	// join adds the index, on the table named table, to its space in r. Call it with
	// the lock of r held, after canJoin.
	join(r *registry, table string)
	// leave removes the index from its space, and drops the space when the index was
	// its last member.
	leave()
}

// uniqueMember is a UniqueIndex of a space, as other indexes of the space reach it.
type uniqueMember[V comparable] interface {
	// holder returns an entry that holds v, with the changes of scope applied.
	holder(ctx context.Context, scope *txState, v V) (Target, bool, error)
	// on reports whether the index is on the table named table.
	on(table string) bool
	// valueOf returns the value of target, an entry of the table of the index, with the
	// changes of scope applied, and false when the table has no entry under its key.
	valueOf(ctx context.Context, scope *txState, target Target) (V, bool, error)
}

// uniqueSpace is the set of UniqueIndexes of one space on a DB.
type uniqueSpace[V comparable] struct {
	// members holds the indexes of the space. A change replaces the slice under the
	// lock of the registry, so a check reads it with no lock.
	members atomic.Pointer[[]uniqueMember[V]]
}
