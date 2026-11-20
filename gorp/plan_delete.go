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
	"cmp"
	"context"
	"slices"
	"strings"
)

// DeleteEffect is what a delete does to one entry that holds an entry it removes.
type DeleteEffect struct {
	// Dependent is the entry that holds Target and the reference it holds it through.
	Dependent
	// Target is the entry that the delete removes.
	Target Target
}

func newDeleteEffect(ref registeredReference, holder, target Target) DeleteEffect {
	return DeleteEffect{
		Dependent: Dependent{
			Holder:    holder,
			Reference: ref.referenceName(),
			Kind:      ref.linkKind(),
			Tag:       ref.referenceTag(),
		},
		Target: target,
	}
}

func compareDeleteEffects(a, b DeleteEffect) int {
	return cmp.Or(
		strings.Compare(a.Holder.Table, b.Holder.Table),
		strings.Compare(a.Holder.Key, b.Holder.Key),
		strings.Compare(a.Reference, b.Reference),
		strings.Compare(a.Target.Table, b.Target.Table),
		strings.Compare(a.Target.Key, b.Target.Key),
	)
}

// DeleteEffects is what a delete would do to the entries that hold the entries it
// removes, through each reference and its OnDelete. Each list is sorted by holder,
// reference, then target.
type DeleteEffects struct {
	// Cascades are the entries that the delete would delete too, each once, with the
	// first target that cascades to it.
	Cascades []DeleteEffect
	// Detaches are the entries that the delete would keep and rewrite, once for each
	// target that they would lose or inline.
	Detaches []DeleteEffect
	// Restricts are the entries that would fail the delete, each once, since it does
	// not remove them.
	Restricts []DeleteEffect
	// Weak are the entries that the delete would keep unchanged, once for each target
	// that they hold through a Weak reference.
	Weak []DeleteEffect
}

// PlanDelete returns what a delete of the entries with keys would do, as seen by snap,
// and writes nothing. It follows cascades as the delete does and lists each Restrict
// holder instead of failing. It fails as the delete does when the delete would change
// more than DB.MaxDeleteSize entries or an index failed to populate.
func (t *Table[K, E]) PlanDelete(
	ctx context.Context,
	snap Snapshot,
	keys []K,
) (DeleteEffects, error) {
	if !t.registered {
		return DeleteEffects{}, nil
	}
	formatted := make([]string, len(keys))
	for i, k := range keys {
		formatted[i] = FormatKey(k)
	}
	w := newPlanWalk(t.name, formatted, t.db.MaxDeleteSize)
	w.effects = make(map[OnDelete][]DeleteEffect)
	if err := t.db.registry.walk(ctx, snap, w); err != nil {
		return DeleteEffects{}, err
	}
	restricts := make([]DeleteEffect, len(w.restrictions))
	for i, rs := range w.restrictions {
		restricts[i] = newDeleteEffect(rs.ref, rs.dependent, rs.target)
	}
	kept := func(effects []DeleteEffect) []DeleteEffect {
		effects = slices.DeleteFunc(effects, func(e DeleteEffect) bool {
			return w.plan.contains(e.Holder)
		})
		slices.SortFunc(effects, compareDeleteEffects)
		return effects
	}
	cascades := w.effects[Cascade]
	slices.SortFunc(cascades, compareDeleteEffects)
	return DeleteEffects{
		Cascades:  cascades,
		Detaches:  kept(append(w.effects[Detach], w.effects[Inline]...)),
		Restricts: kept(restricts),
		Weak:      kept(w.effects[Weak]),
	}, nil
}
