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
	"strconv"

	"github.com/synnaxlabs/gorp/internal/key"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/validate"
)

// register validates the table's references and adds the table to its DB's registry.
// A table with no references joins the registry only when named is true.
func (t *Table[K, E]) register(named bool) error {
	if !named && len(t.references) == 0 {
		return nil
	}
	names := make(set.Set[string], len(t.references))
	for i, r := range t.references {
		err := r.validate(t.name)
		if err == nil && names.Contains(r.referenceName()) {
			err = validate.PathedError(
				errors.Wrap(
					validate.ErrValidation,
					"names another reference of the table",
				),
				"name",
			)
		}
		if err != nil {
			return validate.PathedError(err, t.name, "references", strconv.Itoa(i))
		}
		names.Add(r.referenceName())
		r.bind(t.name, i)
	}
	if err := t.db.registry.register(t.name, t.keyPrefix, t); err != nil {
		return err
	}
	t.registered = true
	return nil
}

// registerReferences adds the table's references to its indexes and to its DB's
// registry. Call it after the table's migrations: a delete that follows a reference
// waits for its index to populate, which starts only after them.
func (t *Table[K, E]) registerReferences() error {
	refs := make([]registeredReference, len(t.references))
	for i, r := range t.references {
		refs[i] = r
		t.indexes = append(t.indexes, r)
	}
	return t.db.registry.addReferences(refs)
}

// unregister removes the table and its references from its DB's registry.
func (t *Table[K, E]) unregister() {
	if !t.registered {
		return
	}
	t.db.registry.unregister(t.name, t.keyPrefix)
	t.registered = false
}

// writer returns a Writer that stages against the indexes of every open table of E on
// the table's key prefix and checks the table's references when checked is true. A
// table outside the registry writes as a Writer with no table does.
func (t *Table[K, E]) writer(tx Tx, checked bool) Writer[K, E] {
	if !t.registered {
		return wrapWriter[K, E](tx, "", t.keyPrefix, nil)
	}
	w := wrapWriter[K, E](tx, t.name, t.keyPrefix, nil)
	w.references = t.references
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
func (t *Table[K, E]) exists(
	ctx context.Context,
	snap Snapshot,
	k string,
) (bool, error) {
	pk, err := key.Parse[K](k)
	if err != nil {
		return false, err
	}
	return t.has(snap, key.NewCodec[K](t.keyPrefix), pk)
}

// has reports whether snap holds the entry with key k, which codec encodes.
func (t *Table[K, E]) has(snap Snapshot, codec key.Codec[K], k K) (bool, error) {
	_, closer, err := snap.Get(codec.Encode(k))
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
	w := t.writer(tx, false)
	w.derived = true
	return w.Delete(ctx, pks...)
}

// issues implements registeredTable.
func (t *Table[K, E]) issues() bool { return t.issuedKey != nil }

// issued implements registeredTable.
func (t *Table[K, E]) issued(k string) (bool, error) {
	pk, err := key.Parse[K](k)
	if err != nil {
		return false, err
	}
	return t.issuedKey(pk), nil
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
	w := t.writer(tx, false)
	w.derived = true
	if err = w.Set(ctx, e); err != nil || t.onRewrite == nil {
		return err
	}
	return t.onRewrite(ctx, tx, e)
}

// delete deletes the entries with keys and applies the OnDelete of every reference
// that points at them, from planned when it is not nil.
func (t *Table[K, E]) delete(
	ctx context.Context,
	tx Tx,
	keys []K,
	planned *DeleteEffects,
) error {
	if t.registered {
		var (
			plan *deletePlan
			err  error
		)
		if planned != nil && current(*planned, tx, t.db.registry, t.name, keys) {
			plan, err = t.db.registry.planned(tx, t.name, *planned)
		} else {
			formatted := make([]string, len(keys))
			for i, k := range keys {
				formatted[i] = FormatKey(k)
			}
			plan, err = t.db.registry.plan(
				ctx,
				tx,
				t.name,
				formatted,
				t.db.MaxDeleteSize,
			)
		}
		if err != nil {
			return err
		}
		if err = t.db.registry.apply(ctx, tx, plan); err != nil {
			return err
		}
	}
	w := t.writer(tx, false)
	return w.Delete(ctx, keys...)
}

// referenceIndex implements registeredTable.
func (t *Table[K, E]) referenceIndex(name string) (int, bool) {
	i := slices.IndexFunc(t.references, func(r Reference[K, E]) bool {
		return r.referenceName() == name
	})
	return i, i >= 0
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

// Dependent is an entry that holds a reference to another entry.
type Dependent struct {
	// Holder is the entry that holds the reference.
	Holder Target
	// Reference is the name of the reference.
	Reference string
	// Kind is the kind of the link.
	Kind LinkKind
	// Tag is the Tag of the reference's config.
	Tag uint8
}

// Dependents returns the entries of every open table that hold a reference to the entry
// with key k, as seen by snap, once for each reference that holds it. A Weak reference
// holds k after k is deleted. It waits for each reference's index to populate, or for
// ctx to end, and returns ErrIndexInvalid if one failed to populate.
func (t *Table[K, E]) Dependents(
	ctx context.Context,
	snap Snapshot,
	k K,
) ([]Dependent, error) {
	deps, err := t.DependentsEach(ctx, snap, []K{k})
	if err != nil {
		return nil, err
	}
	return deps[0], nil
}

// Uses returns the entries of every open table that hold a Use reference to the entry
// with key k, as seen by snap. It waits and fails as Dependents does.
func (t *Table[K, E]) Uses(ctx context.Context, snap Snapshot, k K) ([]Target, error) {
	deps, err := t.Dependents(ctx, snap, k)
	if err != nil {
		return nil, err
	}
	var out []Target
	for _, d := range deps {
		if d.Kind == Use {
			out = append(out, d.Holder)
		}
	}
	return out, nil
}

// PlacedIn returns the records that hold a Place link to target, as seen by tx. It
// waits for each index as Table.Dependents does.
func PlacedIn(ctx context.Context, tx Tx, target Target) ([]Target, error) {
	var out []Target
	for _, ref := range txDB(tx).registry.referencesTo(target.Table) {
		if ref.linkKind() != Place {
			continue
		}
		keys, err := ref.dependentsOf(ctx, tx, target)
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
		keys, err := ref.dependentsOf(ctx, snap, target)
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

// DependentsEach returns, at each position of keys, the dependents of the entry with
// that key, as Dependents does. It probes each reference once for all of keys.
func (t *Table[K, E]) DependentsEach(
	ctx context.Context,
	snap Snapshot,
	keys []K,
) ([][]Dependent, error) {
	targets := make([]Target, len(keys))
	for i, k := range keys {
		targets[i] = Target{Table: t.name, Key: FormatKey(k)}
	}
	return t.db.registry.dependents(ctx, snap, t.name, targets, keys)
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
