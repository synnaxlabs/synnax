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

	"github.com/synnaxlabs/gorp/internal/key"
	"github.com/synnaxlabs/x/set"
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
	// tx is the transaction that planned the delete, nil for a snapshot.
	tx *txState
	// seq is the number of writes that tx had made when it planned the delete.
	seq uint64
	// table is the name of the table that the delete removes entries from.
	table string
	// keys holds, as a set.Set[K], the keys of the entries that the delete removes.
	keys any
}

// current reports whether e still describes the delete of the entries with keys from
// the table named table in tx. It holds when e planned that delete in tx, and no write
// of tx after the plan deleted a Cascade holder of e, left a holder of e in place, or
// left a link to an entry that the delete removes.
func current[K Key](e DeleteEffects, tx Tx, table string, keys []K) bool {
	st := tx.txIdentity()
	if st == nil || st != e.tx || e.table != table {
		return false
	}
	if planned, ok := e.keys.(set.Set[K]); !ok || !sameKeys(planned, keys) {
		return false
	}
	var removed, held set.Set[Target]
	for _, c := range st.changes.entries {
		if c.seq <= e.seq {
			continue
		}
		if !c.present {
			if slices.ContainsFunc(e.Cascades, func(d DeleteEffect) bool {
				return d.Holder == c.record
			}) {
				return false
			}
			continue
		}
		if removed == nil {
			removed, held = touched(e, table, keys)
		}
		if held.Contains(c.record) || slices.ContainsFunc(c.after, func(l Link) bool {
			return removed.Contains(l.Target)
		}) {
			return false
		}
	}
	return true
}

// touched returns the entries that the delete of keys from table with e removes, and
// the holders that e names.
func touched[K Key](
	e DeleteEffects,
	table string,
	keys []K,
) (removed, held set.Set[Target]) {
	removed = make(set.Set[Target], len(keys)+len(e.Cascades))
	for _, k := range keys {
		removed.Add(Target{Table: table, Key: FormatKey(k)})
	}
	held = make(set.Set[Target], len(e.Cascades)+len(e.Detaches)+len(e.Restricts))
	for _, effects := range [][]DeleteEffect{e.Cascades, e.Detaches, e.Restricts} {
		for _, d := range effects {
			held.Add(d.Holder)
		}
	}
	for _, d := range e.Cascades {
		removed.Add(d.Holder)
	}
	return removed, held
}

// sameKeys reports whether planned and keys hold the same keys.
func sameKeys[K Key](planned set.Set[K], keys []K) bool {
	for _, k := range keys {
		if !planned.Contains(k) {
			return false
		}
	}
	for k := range planned {
		if !slices.Contains(keys, k) {
			return false
		}
	}
	return true
}

// PlanDelete returns what a delete of the entries with keys would do, as seen by snap,
// and writes nothing. It plans the keys that exist, once each, as the delete does, and
// follows its cascades, but it lists each Restrict holder instead of failing. It reads
// only what snap holds, so it leaves out a commit in flight. It fails as the delete
// does when the delete would change more than DB.MaxDeleteSize entries or an index
// failed to populate, and with a validation error when more than DB.MaxDeleteSize
// entries would refuse the delete.
func (t *Table[K, E]) PlanDelete(
	ctx context.Context,
	snap Snapshot,
	keys []K,
) (DeleteEffects, error) {
	if !t.registered {
		return DeleteEffects{}, nil
	}
	var (
		codec     = key.NewCodec[K](t.keyPrefix)
		formatted = make([]string, 0, len(keys))
		// existing holds each key whose entry exists.
		existing = make(set.Set[K], len(keys))
	)
	for _, k := range keys {
		if existing.Contains(k) {
			continue
		}
		held, err := t.has(snap, codec, k)
		if err != nil {
			return DeleteEffects{}, err
		}
		if held {
			existing.Add(k)
			formatted = append(formatted, FormatKey(k))
		}
	}
	if len(formatted) == 0 {
		return DeleteEffects{}, nil
	}
	w := newPlanWalk(t.name, formatted, t.db.MaxDeleteSize)
	w.effects = new([Weak + 1][]DeleteEffect)
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
	effects := DeleteEffects{
		Cascades:  lowestPerHolder(w.effects[Cascade]),
		Detaches:  kept(append(w.effects[Detach], w.effects[Inline]...)),
		Restricts: lowestPerHolder(kept(w.effects[Restrict])),
		Weak:      kept(w.effects[Weak]),
	}
	if st := snap.txIdentity(); st != nil {
		effects.tx, effects.seq = st, st.changes.seq
		effects.table, effects.keys = t.name, existing
	}
	return effects, nil
}

// lowestPerHolder sorts effects and keeps the first effect of each holder.
func lowestPerHolder(effects []DeleteEffect) []DeleteEffect {
	slices.SortFunc(effects, compareDeleteEffects)
	return slices.CompactFunc(effects, func(a, b DeleteEffect) bool {
		return a.Holder == b.Holder
	})
}
