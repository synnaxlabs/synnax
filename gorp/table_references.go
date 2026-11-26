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
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/validate"
)

// register validates the table's references and adds the table to its DB's registry.
// A table with no references and no unique index joins the registry only when named is
// true.
func (t *Table[K, E]) register(named bool) error {
	if !named && len(t.references) == 0 && len(t.uniques) == 0 {
		return nil
	}
	for i, r := range t.references {
		if err := r.validate(t.name); err != nil {
			return validate.PathedError(err, t.name, "references", strconv.Itoa(i))
		}
		r.bind(t.name, i)
	}
	if err := t.db.registry.register(t.name, t.keyPrefix, t); err != nil {
		return err
	}
	t.registered = true
	return nil
}

// registerIndexes adds the table's unique indexes to their spaces, and then its
// references to its indexes and to its DB's registry. A failed join adds no reference,
// so no delete waits for the reference index of a table that failed to open. Call it
// after the table's migrations: a delete that follows a reference waits for its index
// to populate, which starts only after them.
func (t *Table[K, E]) registerIndexes() error {
	if err := t.joinSpaces(); err != nil {
		return err
	}
	refs := make([]registeredReference, len(t.references))
	for i, r := range t.references {
		refs[i] = r
		t.indexes = append(t.indexes, r)
	}
	t.db.registry.addReferences(refs)
	return nil
}

// joinSpaces adds the table's unique indexes to their spaces, all of them or none, so
// that no check waits for the index of a table that failed to open.
func (t *Table[K, E]) joinSpaces() error {
	r := t.db.registry
	r.mu.Lock()
	defer r.mu.Unlock()
	spaces := make(set.Set[string], len(t.uniques))
	for _, u := range t.uniques {
		if spaces.Contains(u.spaceName()) {
			return errors.Newf(
				"[gorp] - table %s already has an index in space %q",
				t.name,
				u.spaceName(),
			)
		}
		spaces.Add(u.spaceName())
		if err := u.canJoin(r); err != nil {
			return err
		}
	}
	for _, u := range t.uniques {
		u.join(r, t.name)
	}
	return nil
}

// unregister removes the table and its references from its DB's registry.
func (t *Table[K, E]) unregister() {
	if !t.registered {
		return
	}
	t.db.registry.unregister(t.name, t.keyPrefix, func(r *registry) {
		for _, u := range t.uniques {
			u.leave(r)
		}
	})
	t.registered = false
}

// writerChecks are the checks that a Writer of a Table runs on each entry it sets.
type writerChecks uint8

const (
	// checkReferences checks the references of each entry.
	checkReferences writerChecks = 1 << iota
	// checkUnique checks the values of the unique indexes of the table.
	checkUnique
	// checkAll runs every check.
	checkAll = checkReferences | checkUnique
)

// writer returns a Writer that stages against the indexes of every open table of E on
// the table's key prefix and runs checks on each entry it sets. A table outside the
// registry writes as a Writer with no table does.
func (t *Table[K, E]) writer(tx Tx, checks writerChecks) Writer[K, E] {
	if !t.registered {
		return wrapWriter[K, E](tx, "", t.keyPrefix, nil)
	}
	w := wrapWriter[K, E](tx, t.name, t.keyPrefix, nil)
	w.references = t.references
	if checks&checkReferences != 0 && len(t.references) > 0 {
		w.table = t
	}
	if checks&checkUnique != 0 {
		w.uniques = t.uniques
	}
	return w
}

// canonical implements registeredTable.
func (t *Table[K, E]) canonical(k string) bool {
	pk, err := key.Parse[K](k)
	return err == nil && key.Format(pk) == k
}

// exists implements registeredTable.
func (t *Table[K, E]) exists(
	ctx context.Context,
	snap Snapshot,
	k string,
) (bool, error) {
	pk, err := key.Parse[K](k)
	if err != nil {
		return false, err
	}
	codec := key.NewCodec[K](t.keyPrefix)
	_, closer, err := snap.Get(codec.Encode(pk))
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

// links implements registeredTable.
func (t *Table[K, E]) links(
	ctx context.Context,
	snap Snapshot,
	k string,
) ([]Link, error) {
	pk, err := key.Parse[K](k)
	if err != nil {
		return nil, err
	}
	var e E
	if err = t.NewRetrieve().
		Where(MatchKeys[K, E](pk)).
		Entry(&e).
		Exec(ctx, snap); err != nil {
		return nil, err
	}
	var out []Link
	for _, r := range t.references {
		out = r.links(out, &e)
	}
	return out, nil
}

// deleteKeys implements registeredTable.
func (t *Table[K, E]) deleteKeys(ctx context.Context, tx Tx, keys []string) error {
	pks, err := parseKeys[K](keys)
	if err != nil {
		return err
	}
	w := t.writer(tx, 0)
	w.derived = true
	return w.Delete(ctx, pks...)
}

// release implements registeredTable.
func (t *Table[K, E]) release(tx Tx, keys []string) error {
	for _, u := range t.uniques {
		if err := u.release(tx, keys); err != nil {
			return err
		}
	}
	return nil
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
		ref := t.references[op.ref]
		if ref.onDeletePolicy() != Inline {
			ref.detachTarget(&e, op.target)
			continue
		}
		if err = ref.inlineTarget(ctx, tx, &e, op.target); err != nil {
			return err
		}
	}
	// A detach moves the values of e, so it checks them.
	w := t.writer(tx, checkUnique)
	w.derived = true
	if err = w.Set(ctx, e); err != nil || t.onRewrite == nil {
		return err
	}
	return t.onRewrite(ctx, tx, e)
}

// delete deletes the entries with keys and applies the OnDelete of every reference
// that points at them.
func (t *Table[K, E]) delete(ctx context.Context, tx Tx, keys []K) error {
	if t.registered {
		formatted := make([]string, len(keys))
		for i, k := range keys {
			formatted[i] = FormatKey(k)
		}
		plan, err := t.db.registry.plan(ctx, tx, t.name, formatted, t.db.MaxDeleteSize)
		if err != nil {
			return err
		}
		if err = t.db.registry.apply(ctx, tx, plan); err != nil {
			return err
		}
	}
	w := t.writer(tx, 0)
	return w.Delete(ctx, keys...)
}

// watches implements registeredTable.
func (t *Table[K, E]) watches(table string) bool {
	_, ok := t.onTargetSet[table]
	return ok
}

// targetSet implements registeredTable.
func (t *Table[K, E]) targetSet(
	ctx context.Context,
	tx Tx,
	k string,
	table string,
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
	return t.onTargetSet[table](ctx, tx, e)
}

// Dependents returns the entries of every open table that hold a reference to the entry
// with key k, as seen by snap. It waits for each reference's index to populate, or for
// ctx to end, and returns ErrIndexInvalid if one failed to populate.
func (t *Table[K, E]) Dependents(
	ctx context.Context,
	snap Snapshot,
	k K,
) ([]Target, error) {
	return t.db.registry.dependents(
		ctx,
		snap,
		Target{Table: t.name, Key: FormatKey(k)},
		func(registeredReference) bool { return true },
	)
}

// Uses returns the entries of every open table that hold a Use reference to the entry
// with key k, as seen by snap. It waits and fails as Dependents does.
func (t *Table[K, E]) Uses(ctx context.Context, snap Snapshot, k K) ([]Target, error) {
	return t.db.registry.dependents(
		ctx,
		snap,
		Target{Table: t.name, Key: FormatKey(k)},
		func(ref registeredReference) bool { return ref.linkKind() == Use },
	)
}

// PlacedIn returns the records that hold a Place link to target, as seen by tx. It
// waits for each index as Table.Dependents does.
func PlacedIn(ctx context.Context, tx Tx, target Target) ([]Target, error) {
	var out []Target
	for _, ref := range txDB(tx).registry.referencesTo(target.Table) {
		if ref.linkKind() != Place {
			continue
		}
		keys, err := ref.dependents(ctx, tx, target)
		if err != nil {
			return nil, err
		}
		for _, k := range keys {
			out = append(out, Target{Table: ref.ownerTable(), Key: k})
		}
	}
	return out, nil
}

// LinksOf returns the links that the record target holds, as seen by tx. It returns an
// error wrapping query.ErrNotFound when the record does not exist.
func LinksOf(ctx context.Context, tx Tx, target Target) ([]Link, error) {
	t, err := txDB(tx).registry.table(target.Table)
	if err != nil {
		return nil, err
	}
	return t.links(ctx, tx, target.Key)
}

// RestrictChange returns a RestrictError for a change to field of the entry with key k
// when an entry holds it through a Restrict reference, as seen by snap. It returns nil
// when no entry does.
func (t *Table[K, E]) RestrictChange(
	ctx context.Context,
	snap Snapshot,
	k K,
	field string,
) error {
	target := Target{Table: t.name, Key: FormatKey(k)}
	for _, ref := range t.db.registry.referencesTo(t.name) {
		if ref.onDeletePolicy() != Restrict {
			continue
		}
		keys, err := ref.dependents(ctx, snap, target)
		if err != nil {
			return err
		}
		if len(keys) > 0 {
			return RestrictError{
				Target:    target,
				Holder:    Target{Table: ref.ownerTable(), Key: keys[0]},
				Reference: ref.referenceName(),
				Field:     field,
			}
		}
	}
	return nil
}

// check validates the references e holds against tx. It reports whether every index
// already holds the targets of e for its entry, with the changes of tx applied.
func (t *Table[K, E]) check(ctx context.Context, tx Tx, e *E) (bool, error) {
	current := true
	for _, r := range t.references {
		refCurrent, err := r.check(ctx, tx, t, e)
		if err != nil {
			return false, err
		}
		current = current && refCurrent
	}
	return current, nil
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
