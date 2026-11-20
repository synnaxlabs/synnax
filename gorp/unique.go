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
	"sync"
	"sync/atomic"

	"github.com/synnaxlabs/gorp/internal/index"
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
}

// Error implements error.
func (e UniqueError) Error() string {
	return fmt.Sprintf(
		"[gorp] - %s %s already holds the value in %s",
		e.Holder.Table,
		e.Holder.Key,
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
	value func(e *E) (V, bool)
	store *index.Lookup[*txState, K, V]
	// members is the space of the index on the DB of its table, and table is the name
	// of that table. Both are set when the table opens.
	members *uniqueSpace[V]
	table   string
}

// NewUniqueIndex constructs a UniqueIndex named name in the space named space. value
// returns the value of an entry, or false to exempt the entry. Every index of a space
// must have the same value type V, and a table can have at most one index in a space.
func NewUniqueIndex[K Key, E Entry[K], V comparable](
	name string,
	space string,
	value func(e *E) (V, bool),
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
			if v, ok := u.value(&entry); ok {
				insert(entry.GorpKey(), v)
			}
		},
		func(err error) { finish(populateErr(u.name, err)) }
}

func (u *UniqueIndex[K, E, V]) set(entry E) {
	if v, ok := u.value(&entry); ok {
		u.store.Set(entry.GorpKey(), v)
		return
	}
	u.store.Delete(entry.GorpKey())
}

func (u *UniqueIndex[K, E, V]) delete(key K) { u.store.Delete(key) }

func (u *UniqueIndex[K, E, V]) stageSet(tx Tx, entry E) {
	if v, ok := u.value(&entry); ok {
		u.store.Stage(tx.txIdentity(), entry.GorpKey(), v)
		return
	}
	u.store.StageDelete(tx.txIdentity(), entry.GorpKey())
}

func (u *UniqueIndex[K, E, V]) stageDelete(tx Tx, key K) {
	u.store.StageDelete(tx.txIdentity(), key)
}

// check returns a UniqueError when another entry of the space holds the value of e in
// tx, unless the entry of e already holds that value in tx.
func (u *UniqueIndex[K, E, V]) check(ctx context.Context, tx Tx, e *E) error {
	v, ok := u.value(e)
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
		return u.clash(NewTarget(u.table, other), err)
	}
	for _, m := range *u.members.load() {
		if m == uniqueMember[V](u) {
			continue
		}
		if holder, found, err := m.holder(ctx, scope, v); err != nil || found {
			return u.clash(holder, err)
		}
	}
	return nil
}

// clash returns err, or else the UniqueError for holder.
func (u *UniqueIndex[K, E, V]) clash(holder Target, err error) error {
	if err != nil {
		return err
	}
	return UniqueError{Space: u.space, Holder: holder}
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

// tableName implements uniqueMember.
func (u *UniqueIndex[K, E, V]) tableName() string { return u.table }

// join implements uniqueIndex.
func (u *UniqueIndex[K, E, V]) join(r *registry, table string) error {
	u.table = table
	members, err := joinSpace[V](r, u.space, u)
	if err != nil {
		return err
	}
	u.members = members
	return nil
}

// leave implements uniqueIndex.
func (u *UniqueIndex[K, E, V]) leave() {
	if u.members != nil {
		u.members.remove(u)
	}
}

// uniqueIndex is the part of a UniqueIndex that its Table reaches.
type uniqueIndex[K Key, E Entry[K]] interface {
	Index[K, E]
	// check returns a UniqueError when another entry holds the value of e in tx.
	check(ctx context.Context, tx Tx, e *E) error
	// join adds the index, on the table named table, to its space in r.
	join(r *registry, table string) error
	// leave removes the index from its space.
	leave()
}

// uniqueMember is a UniqueIndex of a space, as other indexes of the space reach it.
type uniqueMember[V comparable] interface {
	// holder returns an entry that holds v, with the changes of scope applied.
	holder(ctx context.Context, scope *txState, v V) (Target, bool, error)
	// tableName returns the name of the table of the index.
	tableName() string
}

// uniqueSpace is the set of UniqueIndexes of one space on a DB.
type uniqueSpace[V comparable] struct {
	// mu orders changes to members.
	mu sync.Mutex
	// members holds the indexes of the space. A change replaces the slice, so a check
	// reads it with no lock.
	members atomic.Pointer[[]uniqueMember[V]]
}

func (s *uniqueSpace[V]) load() *[]uniqueMember[V] { return s.members.Load() }

func (s *uniqueSpace[V]) remove(m uniqueMember[V]) {
	s.mu.Lock()
	defer s.mu.Unlock()
	kept := slices.DeleteFunc(
		slices.Clone(*s.members.Load()),
		func(o uniqueMember[V]) bool { return o == m },
	)
	s.members.Store(&kept)
}

// joinSpace adds m to the space named name in r, creating the space on first use. It
// fails when the space holds values of another type, or an index of the table of m.
func joinSpace[V comparable](
	r *registry,
	name string,
	m uniqueMember[V],
) (*uniqueSpace[V], error) {
	r.mu.Lock()
	found, ok := r.spaces[name]
	if !ok {
		space := &uniqueSpace[V]{}
		space.members.Store(&[]uniqueMember[V]{})
		r.spaces[name], found = space, space
	}
	r.mu.Unlock()
	space, ok := found.(*uniqueSpace[V])
	if !ok {
		return nil, errors.Newf(
			"[gorp] - space %q holds values of another type than %T",
			name,
			*new(V),
		)
	}
	space.mu.Lock()
	defer space.mu.Unlock()
	members := *space.members.Load()
	if slices.ContainsFunc(members, func(o uniqueMember[V]) bool {
		return o.tableName() == m.tableName()
	}) {
		return nil, errors.Newf(
			"[gorp] - table %s already has an index in space %q",
			m.tableName(),
			name,
		)
	}
	joined := append(slices.Clone(members), m)
	space.members.Store(&joined)
	return space, nil
}
