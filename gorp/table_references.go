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

// registerIndexes adds the table's references to its indexes and to its DB's
// registry, and its unique indexes to their spaces. Call it after the table's
// migrations: a delete that follows a reference waits for its index to populate, which
// starts only after them.
func (t *Table[K, E]) registerIndexes() error {
	refs := make([]registeredReference, len(t.references))
	for i, r := range t.references {
		refs[i] = r
		t.indexes = append(t.indexes, r)
	}
	if err := t.db.registry.addReferences(refs); err != nil {
		return err
	}
	return t.joinSpaces()
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
	for _, u := range t.uniques {
		u.leave()
	}
	t.db.registry.unregister(t.name, t.keyPrefix)
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

// formatKey implements registeredTable.
func (t *Table[K, E]) formatKey(k string) (string, error) {
	pk, err := key.Parse[K](k)
	if err != nil {
		return "", err
	}
	return key.Format(pk), nil
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
	// A detach moves the values of e, so it checks them.
	w := t.writer(tx, checkUnique)
	w.derived = true
	if err = w.Set(ctx, e); err != nil || t.onRewrite == nil {
		return err
	}
	return t.onRewrite(ctx, tx, e)
}

// retarget implements registeredTable.
func (t *Table[K, E]) retarget(
	ctx context.Context,
	tx Tx,
	k string,
	refs []string,
	table string,
	to any,
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
	moved := false
	for _, name := range refs {
		i := slices.IndexFunc(t.references, func(r Reference[K, E]) bool {
			return r.referenceName() == name
		})
		if i < 0 {
			return errors.Newf("[gorp] - table %s has no reference %s", t.name, name)
		}
		held, err := t.references[i].retargetTargets(&e, table, to)
		if err != nil {
			return err
		}
		moved = moved || held
	}
	if !moved {
		return nil
	}
	w := t.writer(tx, checkAll)
	if err = w.Set(ctx, e); err != nil || t.onRewrite == nil {
		return err
	}
	return t.onRewrite(ctx, tx, e)
}

// Retarget moves the links that holders keep to entries of the table: each named
// reference of each holder replaces every target that is a key of to with its value.
// holders are dependents that Dependents returns, or a subset of them. Gorp applies to
// once per holder, so a map that swaps two keys swaps them, and writes each holder that
// held a key of to once, through its reference checks, then runs OnRewrite. A Unique
// reference checks each holder against the others as they stand, so a map cannot swap
// or chain targets between holders of one Unique reference.
//
// Retarget fails with a validation error when to moves the zero key or moves a key onto
// it. It fails on a holder that does not exist, and on a reference that has no Retarget
// or does not point at the table. It can fail after it writes other holders, so discard
// tx on an error.
func (t *Table[K, E]) Retarget(
	ctx context.Context,
	tx Tx,
	to map[K]K,
	holders []Dependent,
) error {
	if len(to) == 0 {
		return nil
	}
	var zero K
	for from, onto := range to {
		if from == zero || onto == zero {
			return errors.Wrapf(
				validate.ErrValidation,
				"cannot retarget %s to or from the zero key",
				t.name,
			)
		}
	}
	var (
		order []Target
		refs  = make(map[Target][]string, len(holders))
	)
	for _, d := range holders {
		table, err := t.db.registry.table(d.Holder.Table)
		if err != nil {
			return err
		}
		h := d.Holder
		if h.Key, err = table.formatKey(h.Key); err != nil {
			return err
		}
		named, ok := refs[h]
		if !ok {
			order = append(order, h)
		}
		if !slices.Contains(named, d.Reference) {
			refs[h] = append(named, d.Reference)
		}
	}
	for _, h := range order {
		table, err := t.db.registry.table(h.Table)
		if err != nil {
			return err
		}
		if err = table.retarget(ctx, tx, h.Key, refs[h], t.name, to); err != nil {
			return err
		}
	}
	return nil
}

// delete deletes entries and applies the OnDelete of every reference that points at
// them, from planned when it is not nil. Before it changes anything, it calls held,
// when not nil, with each entry it removes from the table and its Weak holders.
func (t *Table[K, E]) delete(
	ctx context.Context,
	tx Tx,
	entries []E,
	planned *DeleteEffects,
	held HeldFunc[K, E],
) error {
	keys := make([]K, len(entries))
	for i, e := range entries {
		keys[i] = e.GorpKey()
	}
	if t.registered {
		var (
			plan *deletePlan
			err  error
		)
		weak := held != nil
		if planned != nil {
			plan, err = t.db.registry.planned(ctx, tx, t.name, *planned, weak)
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
				weak,
			)
		}
		if err != nil {
			return err
		}
		if held != nil {
			if err = t.callHeld(ctx, tx, entries, plan, held); err != nil {
				return err
			}
		}
		if err = t.db.registry.apply(ctx, tx, plan); err != nil {
			return err
		}
	}
	w := t.writer(tx, 0)
	return w.Delete(ctx, keys...)
}

// callHeld calls held with each entry of the table that the delete of plan removes, and
// with the Weak holders of each: the entries the caller asked to delete first, then, in
// key order, those that a cascade deletes.
func (t *Table[K, E]) callHeld(
	ctx context.Context,
	tx Tx,
	entries []E,
	plan *deletePlan,
	held HeldFunc[K, E],
) error {
	holders := func(e E) []Dependent {
		if len(plan.weak) == 0 {
			return nil
		}
		return plan.weak[FormatKey(e.GorpKey())]
	}
	for _, e := range entries {
		if err := held(ctx, tx, e, holders(e)); err != nil {
			return err
		}
	}
	cascaded := plan.cascades[t.name]
	if len(cascaded) == 0 {
		return nil
	}
	texts := cascaded.Slice()
	slices.Sort(texts)
	keys := make([]K, len(texts))
	for i, text := range texts {
		k, err := key.Parse[K](text)
		if err != nil {
			return err
		}
		keys[i] = k
	}
	var more []E
	if err := t.NewRetrieve().
		Where(MatchKeys[K, E](keys...)).
		Entries(&more).
		Exec(ctx, tx); err != nil {
		return err
	}
	for _, e := range more {
		if err := held(ctx, tx, e, holders(e)); err != nil {
			return err
		}
	}
	return nil
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

// ReferenceID returns the ID of the reference through which d holds its target.
func (d Dependent) ReferenceID() ReferenceID {
	return ReferenceID{Table: d.Holder.Table, Name: d.Reference}
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
