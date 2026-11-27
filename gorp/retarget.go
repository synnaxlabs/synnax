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
	"strings"

	"github.com/synnaxlabs/gorp/internal/key"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/validate"
)

// Retarget moves every reference that points at an entry whose key moves holds, so that
// it points at the entry that moves maps the key to, through the Retarget of each
// reference, in one transaction. Every move applies at once, so two entries can swap
// their holders. It writes each holder with a checked write and then runs the holder
// table's OnRewrite. It leaves every entry in place, so the caller can delete the old
// ones next.
//
// Retarget fails with a validation error when a key moves to itself, when a reference
// that holds a moved key has no Retarget, when a holder's key would change, when it
// would rewrite more entries than the DB's MaxDeleteSize, or when a rewritten holder
// fails its write check. It fails with query.ErrNotFound when a key or its target does
// not exist. It waits for each reference's index to populate, or for ctx to end, and
// fails with ErrIndexInvalid if one failed to populate.
func (t *Table[K, E]) Retarget(ctx context.Context, tx Tx, moves map[K]K) error {
	checkForNilTx("Table.Retarget", tx)
	froms := make([]Target, 0, len(moves))
	targets := make(map[Target]Target, len(moves))
	for from, to := range moves {
		if from == to {
			return errors.Wrapf(
				validate.ErrValidation,
				"cannot retarget %s %s to itself",
				t.name,
				FormatKey(from),
			)
		}
		f := NewTarget(t.name, from)
		froms = append(froms, f)
		targets[f] = NewTarget(t.name, to)
	}
	slices.SortFunc(froms, func(a, b Target) int {
		return strings.Compare(a.Key, b.Key)
	})
	return inTx(ctx, tx, func(tx Tx) error {
		for _, f := range froms {
			for _, k := range []string{f.Key, targets[f].Key} {
				ok, err := t.exists(ctx, tx, k)
				if err != nil {
					return err
				}
				if !ok {
					return errors.Wrapf(
						query.ErrNotFound,
						"%s %s does not exist",
						t.name,
						k,
					)
				}
			}
		}
		return t.db.registry.retarget(ctx, tx, froms, targets, t.db.MaxDeleteSize)
	})
}

// retarget implements registeredTable.
func (t *Table[K, E]) retarget(
	ctx context.Context,
	tx Tx,
	k string,
	refs []int,
	from Target,
	moves map[Target]Target,
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
	for _, ref := range refs {
		t.references[ref].retargetTargets(&e, moves)
	}
	if e.GorpKey() != pk {
		return errors.Wrapf(
			validate.ErrValidation,
			"cannot retarget %s %s because the key of %s %s holds it",
			from.Table,
			from.Key,
			t.name,
			k,
		)
	}
	w := t.writer(tx, true)
	w.derived = true
	if err = w.Set(ctx, e); err != nil || t.onRewrite == nil {
		return err
	}
	return t.onRewrite(ctx, tx, e)
}

// retargetHolder is an entry that holds a moved target, with the first target that it
// holds and the indexes of the references through which it holds a moved target.
type retargetHolder struct {
	entry Target
	from  Target
	refs  []int
}

// retarget moves every holder of each of froms to the target that moves maps it to. It
// finds every holder before it rewrites one, so a reference with no Retarget fails the
// call before it changes anything, and it rewrites each holder once. It fails when it
// would rewrite more than limit entries.
func (r *registry) retarget(
	ctx context.Context,
	tx Tx,
	froms []Target,
	moves map[Target]Target,
	limit int,
) error {
	var (
		holders []retargetHolder
		indexOf = make(map[Target]int)
	)
	for _, from := range froms {
		for _, ref := range r.referencesTo(from.Table) {
			deps, err := ref.dependents(ctx, tx, from)
			if err != nil {
				return err
			}
			for _, k := range deps {
				dep := Target{Table: ref.ownerTable(), Key: k}
				if !ref.canRetarget() {
					return errors.Wrapf(
						validate.ErrValidation,
						"cannot retarget %s %s because %s %s references it through "+
							"%s, which cannot retarget",
						from.Table,
						from.Key,
						dep.Table,
						dep.Key,
						ref.referenceName(),
					)
				}
				i, ok := indexOf[dep]
				if !ok {
					i = len(holders)
					indexOf[dep] = i
					holders = append(holders, retargetHolder{entry: dep, from: from})
					if len(holders) > limit {
						return errors.Wrapf(
							validate.ErrValidation,
							"retargeting %s %s would change more than %d entries",
							from.Table,
							from.Key,
							limit,
						)
					}
				}
				if !slices.Contains(holders[i].refs, ref.index()) {
					holders[i].refs = append(holders[i].refs, ref.index())
				}
			}
		}
	}
	for _, h := range holders {
		table, err := r.table(h.entry.Table)
		if err != nil {
			return err
		}
		if err = table.retarget(
			ctx,
			tx,
			h.entry.Key,
			h.refs,
			h.from,
			moves,
		); err != nil {
			return err
		}
	}
	return nil
}

// RetargetEach returns s with each element whose target to holds as a key pointed at
// the target that to maps it to, through point. target returns the target of an
// element. A moved element whose new target another element holds is dropped, so a
// move never adds a repeat. It returns s itself when no element moves.
func RetargetEach[E any, T comparable](
	s []E,
	to map[T]T,
	target func(E) T,
	point func(e *E, t T),
) []E {
	held := make(set.Set[T], len(s))
	moved := false
	for _, e := range s {
		if _, ok := to[target(e)]; ok {
			moved = true
		} else {
			held.Add(target(e))
		}
	}
	if !moved {
		return s
	}
	out := make([]E, 0, len(s))
	for _, e := range s {
		t, ok := to[target(e)]
		if !ok {
			out = append(out, e)
			continue
		}
		if held.Contains(t) {
			continue
		}
		held.Add(t)
		point(&e, t)
		out = append(out, e)
	}
	return out
}
