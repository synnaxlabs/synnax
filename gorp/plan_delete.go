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

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
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
		Holder:    holder,
		Reference: ref.referenceName(),
		Kind:      ref.linkKind(),
		Tag:       ref.referenceTag(),
		Target:    target,
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
// removes, through each reference and its OnDelete. Cascades and Restricts list each
// holder once, with its lowest reference and target; Detaches and Weak list a holder
// once for each target, so one holder can be in both. A holder that the delete removes
// is only in Cascades. Each list is sorted by holder, reference, then target.
type DeleteEffects struct {
	// Cascades are the entries that the delete would delete too.
	Cascades []DeleteEffect
	// Detaches are the entries that the delete would keep and rewrite, losing or
	// inlining a target.
	Detaches []DeleteEffect
	// Restricts are the entries that would fail the delete, since it does not remove
	// them.
	Restricts []DeleteEffect
	// Weak are the entries that the delete would keep unchanged, holding a target
	// through a Weak reference.
	Weak []DeleteEffect
}

// PlanDelete returns what a delete of the entries with keys would do, as seen by snap,
// and writes nothing. It plans the keys that exist, once each, as the delete does, and
// follows its cascades, but it lists each Restrict holder instead of failing. Unlike
// the delete, it leaves out the entries of a commit in flight that snap does not hold.
// It fails as the delete does when the delete would change more than DB.MaxDeleteSize
// entries or an index failed to populate, and with a validation error when more than
// DB.MaxDeleteSize entries would refuse the delete.
func (t *Table[K, E]) PlanDelete(
	ctx context.Context,
	snap Snapshot,
	keys []K,
) (DeleteEffects, error) {
	if !t.registered {
		return DeleteEffects{}, nil
	}
	var entries []E
	if err := t.NewRetrieve().
		Where(MatchKeys[K, E](keys...)).
		Entries(&entries).
		Exec(ctx, snap); err != nil && !errors.Is(err, query.ErrNotFound) {
		return DeleteEffects{}, err
	}
	if len(entries) == 0 {
		return DeleteEffects{}, nil
	}
	formatted := make([]string, len(entries))
	for i, e := range entries {
		formatted[i] = FormatKey(e.GorpKey())
	}
	w := newPlanWalk(t.name, formatted, t.db.MaxDeleteSize)
	w.effects = make(map[OnDelete][]DeleteEffect)
	if err := t.db.registry.walk(ctx, snap, w); err != nil {
		return DeleteEffects{}, err
	}
	kept := func(effects []DeleteEffect) []DeleteEffect {
		effects = slices.DeleteFunc(effects, func(e DeleteEffect) bool {
			return w.plan.contains(e.Holder)
		})
		slices.SortFunc(effects, compareDeleteEffects)
		return effects
	}
	return DeleteEffects{
		Cascades:  lowestPerHolder(w.effects[Cascade]),
		Detaches:  kept(append(w.effects[Detach], w.effects[Inline]...)),
		Restricts: lowestPerHolder(kept(w.effects[Restrict])),
		Weak:      kept(w.effects[Weak]),
	}, nil
}

// lowestPerHolder sorts effects and keeps the first effect of each holder.
func lowestPerHolder(effects []DeleteEffect) []DeleteEffect {
	slices.SortFunc(effects, compareDeleteEffects)
	return slices.CompactFunc(effects, func(a, b DeleteEffect) bool {
		return a.Holder == b.Holder
	})
}
