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
	"strconv"

	"github.com/synnaxlabs/gorp/internal/key"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
)

// register validates the table's references and adds the table to its DB's registry.
// A table with no references joins the registry only when named is true.
func (t *Table[K, E]) register(named bool) error {
	if !named && len(t.references) == 0 {
		return nil
	}
	for i, r := range t.references {
		if err := r.validate(t.name); err != nil {
			return validate.PathedError(err, t.name, "references", strconv.Itoa(i))
		}
		r.bind(t.name, i)
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
	if err = closer.Close(); err != nil {
		return false, err
	}
	return true, nil
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
		t.references[op.ref].detachTarget(&e, op.target)
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
	plan, err := t.db.registry.plan(ctx, tx, t.name, formatted, t.db.MaxDeleteSize)
	if err != nil {
		return err
	}
	return t.db.registry.apply(ctx, tx, plan)
}

// Dependents returns the entries of every open table that hold a reference to the entry
// with key k, as seen by tx. It waits for each reference's index to populate, or for
// ctx to end, and returns ErrIndexInvalid if one failed to populate.
func (t *Table[K, E]) Dependents(ctx context.Context, tx Tx, k K) ([]Target, error) {
	return t.db.registry.dependents(ctx, tx, Target{Table: t.name, Key: FormatKey(k)})
}

// check validates the references e holds against tx.
func (t *Table[K, E]) check(ctx context.Context, tx Tx, e *E) error {
	for _, r := range t.references {
		if err := r.check(ctx, tx, t, e); err != nil {
			return err
		}
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
