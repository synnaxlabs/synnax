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

	"github.com/synnaxlabs/gorp/internal/key"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
)

// Retarget moves every reference that points at the entry with key from so that it
// points at the entry with key to, through the Retarget of each reference, in one
// transaction. It writes each holder with a checked write and then runs the holder
// table's OnRewrite. It leaves both entries in place, so the caller can delete from
// next.
//
// Retarget fails with a validation error when from equals to, when a reference that
// holds from has no Retarget, when a holder's key would change, when it would rewrite
// more entries than the DB's MaxDeleteSize, or when a rewritten holder fails its write
// check. It fails with query.ErrNotFound when from or to does not exist. It waits for
// each reference's index to populate, or for ctx to end, and fails with ErrIndexInvalid
// if one failed to populate.
func (t *Table[K, E]) Retarget(ctx context.Context, tx Tx, from, to K) error {
	checkForNilTx("Table.Retarget", tx)
	if from == to {
		return errors.Wrapf(
			validate.ErrValidation,
			"cannot retarget %s %s to itself",
			t.name,
			FormatKey(from),
		)
	}
	return inTx(ctx, tx, func(tx Tx) error {
		for _, k := range []K{from, to} {
			ok, err := t.exists(ctx, tx, FormatKey(k))
			if err != nil {
				return err
			}
			if !ok {
				return errors.Wrapf(
					query.ErrNotFound,
					"%s %s does not exist",
					t.name,
					FormatKey(k),
				)
			}
		}
		return t.db.registry.retarget(
			ctx,
			tx,
			NewTarget(t.name, from),
			NewTarget(t.name, to),
			t.db.MaxDeleteSize,
		)
	})
}

// retarget implements registeredTable.
func (t *Table[K, E]) retarget(
	ctx context.Context,
	tx Tx,
	k string,
	refs []int,
	from, to Target,
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
		t.references[ref].retargetTarget(&e, from, to)
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

// retargetHolder is an entry that holds the target of a retarget, with the indexes of
// the references through which it holds it.
type retargetHolder struct {
	entry Target
	refs  []int
}

// retarget moves every holder of from to to. It finds every holder before it rewrites
// one, so a reference with no Retarget fails the call before it changes anything. It
// fails when it would rewrite more than limit entries.
func (r *registry) retarget(
	ctx context.Context,
	tx Tx,
	from, to Target,
	limit int,
) error {
	var (
		holders []retargetHolder
		indexOf = make(map[Target]int)
	)
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
					"cannot retarget %s %s because %s %s references it through %s, "+
						"which cannot retarget",
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
				holders = append(holders, retargetHolder{entry: dep})
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
			holders[i].refs = append(holders[i].refs, ref.index())
		}
	}
	for _, h := range holders {
		table, err := r.table(h.entry.Table)
		if err != nil {
			return err
		}
		if err = table.retarget(ctx, tx, h.entry.Key, h.refs, from, to); err != nil {
			return err
		}
	}
	return nil
}
