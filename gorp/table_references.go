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
	"slices"

	"github.com/synnaxlabs/gorp/internal/key"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/set"
)

// register validates the table's references and adds the table to its DB's registry.
// A table with no references joins the registry only when named is true.
func (t *Table[K, E]) register(named bool) error {
	if !named && len(t.references) == 0 {
		return nil
	}
	for i, r := range t.references {
		if err := r.validate(t.name); err != nil {
			return err
		}
		r.owner, r.position = t.name, i
	}
	if err := t.db.registry.register(t.name, t); err != nil {
		return err
	}
	t.registered = true
	return nil
}

// registerReferences adds the table's references to its indexes and to its DB's
// registry. Call it after the table's migrations: a delete that follows a reference
// waits for its index to populate, which starts only after them.
func (t *Table[K, E]) registerReferences() {
	refs := make([]registeredReference, len(t.references))
	for i, r := range t.references {
		refs[i] = r
		t.indexes = append(t.indexes, r)
	}
	t.db.registry.addReferences(refs)
}

// unregister removes the table and its references from its DB's registry.
func (t *Table[K, E]) unregister() {
	if !t.registered {
		return
	}
	t.db.registry.unregister(t.name)
	t.registered = false
}

// writer returns a Writer that stages against the table's indexes and checks the
// table's references when checked is true.
func (t *Table[K, E]) writer(tx Tx, checked bool) Writer[K, E] {
	w := wrapWriter[K, E](tx, t.keyPrefix, t.indexes)
	if checked && len(t.references) > 0 {
		w.table = t
	}
	return w
}

// canonical implements registeredTable.
func (t *Table[K, E]) canonical(k string) bool {
	pk, err := key.Parse[K](k)
	return err == nil && key.Format(pk) == k
}

// exists implements registeredTable.
func (t *Table[K, E]) exists(ctx context.Context, tx Tx, k string) (bool, error) {
	pk, err := key.Parse[K](k)
	if err != nil {
		return false, err
	}
	codec := key.NewCodec[K](t.keyPrefix)
	_, closer, err := tx.Get(ctx, codec.Encode(pk))
	if errors.Is(err, query.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, closer.Close()
}

// deleteKeys implements registeredTable.
func (t *Table[K, E]) deleteKeys(ctx context.Context, tx Tx, keys []string) error {
	pks, err := parseKeys[K](keys)
	if err != nil {
		return err
	}
	w := t.writer(tx, false)
	return w.Delete(ctx, pks...)
}

// detach implements registeredTable.
func (t *Table[K, E]) detach(
	ctx context.Context,
	tx Tx,
	k string,
	ops []detachOp,
) error {
	pk, err := key.Parse[K](k)
	if err != nil {
		return err
	}
	var e E
	if err = t.NewRetrieve().
		Where(MatchKeys[K, E](pk)).
		Entry(&e).
		Exec(ctx, tx); err != nil {
		return err
	}
	for _, op := range ops {
		t.references[op.ref].cfg.Detach(&e, op.target)
	}
	w := t.writer(tx, false)
	return w.Set(ctx, e)
}

// delete deletes the entries with keys and applies the OnDelete of every reference
// that points at them.
func (t *Table[K, E]) delete(ctx context.Context, tx Tx, keys []K) error {
	if !t.registered {
		w := t.writer(tx, false)
		return w.Delete(ctx, keys...)
	}
	formatted := make([]string, len(keys))
	for i, k := range keys {
		formatted[i] = FormatKey(k)
	}
	plan, err := t.db.registry.plan(tx, t.name, formatted, t.db.MaxDeleteSize)
	if err != nil {
		return err
	}
	return t.db.registry.apply(ctx, tx, plan)
}

// Dependents returns the entries of every open table that hold a reference to the entry
// with key k, as seen by tx.
func (t *Table[K, E]) Dependents(tx Tx, k K) ([]Target, error) {
	return t.db.registry.dependents(tx, Target{Table: t.name, Key: FormatKey(k)})
}

// check validates the references e holds against tx.
func (t *Table[K, E]) check(ctx context.Context, tx Tx, e *E) error {
	for _, r := range t.references {
		targets := r.cfg.Targets(e)
		for _, target := range targets {
			if !slices.Contains(r.cfg.Tables, target.Table) {
				return r.fieldErr(
					"%s cannot reference %s %s",
					t.name,
					target.Table,
					target.Key,
				)
			}
			table, err := t.db.registry.table(target.Table)
			if err != nil {
				return err
			}
			if !table.canonical(target.Key) {
				return r.fieldErr("%q is not a key of %s", target.Key, target.Table)
			}
			ok, err := table.exists(ctx, tx, target.Key)
			if err != nil {
				return err
			}
			if !ok {
				return r.fieldErr("%s %s does not exist", target.Table, target.Key)
			}
		}
		if r.cfg.Unique {
			if err := t.checkUnique(tx, r, e, targets); err != nil {
				return err
			}
		}
		if r.cfg.Acyclic {
			if err := t.checkAcyclic(ctx, tx, r, e, targets); err != nil {
				return err
			}
		}
	}
	return nil
}

func (t *Table[K, E]) checkUnique(
	tx Tx,
	r *Reference[K, E],
	e *E,
	targets []Target,
) error {
	self := (*e).GorpKey()
	for _, target := range targets {
		holders, err := r.Get(tx, target)
		if err != nil {
			return err
		}
		for _, h := range holders {
			if h != self {
				return r.fieldErr(
					"%s %s already references %s %s",
					t.name,
					FormatKey(h),
					target.Table,
					target.Key,
				)
			}
		}
	}
	return nil
}

// checkAcyclic walks the chain of targets in the table from e and fails if the chain
// reaches e.
func (t *Table[K, E]) checkAcyclic(
	ctx context.Context,
	tx Tx,
	r *Reference[K, E],
	e *E,
	targets []Target,
) error {
	self := Target{Table: t.name, Key: FormatKey((*e).GorpKey())}
	visited := set.New[Target]()
	frontier := slices.Clone(targets)
	for len(frontier) > 0 {
		next := frontier[len(frontier)-1]
		frontier = frontier[:len(frontier)-1]
		if next.Table != t.name || visited.Contains(next) {
			continue
		}
		if next == self {
			return r.fieldErr("%s %s would form a cycle", t.name, self.Key)
		}
		visited.Add(next)
		pk, err := key.Parse[K](next.Key)
		if err != nil {
			return err
		}
		var parent E
		err = t.NewRetrieve().Where(MatchKeys[K, E](pk)).Entry(&parent).Exec(ctx, tx)
		if errors.Is(err, query.ErrNotFound) {
			continue
		}
		if err != nil {
			return err
		}
		frontier = append(frontier, r.cfg.Targets(&parent)...)
	}
	return nil
}

func parseKeys[K Key](keys []string) ([]K, error) {
	out := make([]K, len(keys))
	for i, k := range keys {
		pk, err := key.Parse[K](k)
		if err != nil {
			return nil, err
		}
		out[i] = pk
	}
	return out, nil
}
