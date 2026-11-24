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
			newMoves(t.name, map[K]K{from: to}),
			t.db.MaxDeleteSize,
		)
	})
}

// RetargetHolders moves the links that holders keep to entries of the table: each named
// reference of each holder replaces every target that is a key of to with its value.
// holders are dependents that Dependents returns, or a subset of them. Gorp applies to
// once per holder, so a map that swaps two keys swaps them. It writes each holder that
// holds a key of to once, with a checked write, then runs the holder table's OnRewrite.
// A Unique reference checks each holder against the others as they stand, so a map
// cannot swap or chain targets between holders of one Unique reference.
//
// RetargetHolders fails with a validation error when to moves the zero key or moves a
// key onto it, when a named reference has no Retarget, when a holder's key would
// change, when it would rewrite more entries than the DB's MaxDeleteSize, or when a
// rewritten holder fails its write check. It fails on a holder that does not exist, and
// on a reference that the holder's table does not have or that does not point at the
// table. It checks every reference before it writes a holder.
func (t *Table[K, E]) RetargetHolders(
	ctx context.Context,
	tx Tx,
	to map[K]K,
	holders ...Dependent,
) error {
	checkForNilTx("Table.RetargetHolders", tx)
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
	if len(holders) == 0 {
		return nil
	}
	m := newMoves(t.name, to)
	return inTx(ctx, tx, func(tx Tx) error {
		return t.db.registry.retargetHolders(
			ctx,
			tx,
			m,
			holders,
			t.db.MaxDeleteSize,
		)
	})
}

// formatKey implements registeredTable.
func (t *Table[K, E]) formatKey(k string) (string, error) {
	pk, err := key.Parse[K](k)
	if err != nil {
		return "", err
	}
	return FormatKey(pk), nil
}

// retarget implements registeredTable.
func (t *Table[K, E]) retarget(
	ctx context.Context,
	tx Tx,
	k string,
	refs []int,
	m moves,
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
	for _, ref := range refs {
		held, err := t.references[ref].retargetTargets(&e, m)
		if err != nil {
			return err
		}
		moved = moved || held
	}
	if !moved {
		return nil
	}
	if e.GorpKey() != pk {
		return errors.Wrapf(
			validate.ErrValidation,
			"cannot retarget %s because the key of %s %s holds it",
			formatSources(m.withTargets().targets),
			t.name,
			k,
		)
	}
	w := t.writer(tx, checkAll)
	w.derived = true
	if err = w.Set(ctx, e); err != nil || t.onRewrite == nil {
		return err
	}
	return t.onRewrite(ctx, tx, e)
}

// retargetHolder is an entry that holds a target of a retarget, with the indexes of the
// references through which it holds one.
type retargetHolder struct {
	entry Target
	refs  []int
}

// retargetPlan collects the holders of one retarget before it rewrites any.
type retargetPlan struct {
	m       moves
	limit   int
	holders []retargetHolder
	indexOf map[Target]int
}

// add records that dep holds a target of the plan through ref. It fails when ref has no
// Retarget, or when the plan would rewrite more than limit entries.
func (p *retargetPlan) add(dep Target, ref registeredReference) error {
	if !ref.canRetarget() {
		return errors.Wrapf(
			validate.ErrValidation,
			"cannot retarget %s because %s %s references it through %s, "+
				"which cannot retarget",
			formatSources(p.m.withTargets().targets),
			dep.Table,
			dep.Key,
			ref.referenceName(),
		)
	}
	i, ok := p.indexOf[dep]
	if !ok {
		i = len(p.holders)
		p.indexOf[dep] = i
		p.holders = append(p.holders, retargetHolder{entry: dep})
		if len(p.holders) > p.limit {
			return errors.Wrapf(
				validate.ErrValidation,
				"retargeting %s would change more than %d entries",
				formatSources(p.m.withTargets().targets),
				p.limit,
			)
		}
	}
	if !slices.Contains(p.holders[i].refs, ref.index()) {
		p.holders[i].refs = append(p.holders[i].refs, ref.index())
	}
	if p.m.targets == nil && !ref.retargetsKeys(p.m) {
		p.m = p.m.withTargets()
	}
	return nil
}

// apply rewrites each holder of the plan in tx.
func (p *retargetPlan) apply(ctx context.Context, tx Tx, r *registry) error {
	for _, h := range p.holders {
		table, err := r.table(h.entry.Table)
		if err != nil {
			return err
		}
		if err = table.retarget(ctx, tx, h.entry.Key, h.refs, p.m); err != nil {
			return err
		}
	}
	return nil
}

// retarget moves every holder of from through m. It finds every holder before it
// rewrites one, so a reference with no Retarget fails the call before it changes
// anything. It fails when it would rewrite more than limit entries.
func (r *registry) retarget(
	ctx context.Context,
	tx Tx,
	from Target,
	m moves,
	limit int,
) error {
	p := retargetPlan{m: m, limit: limit, indexOf: make(map[Target]int)}
	for _, ref := range r.referencesTo(from.Table) {
		deps, err := ref.dependentsOf(ctx, tx, from)
		if err != nil {
			return err
		}
		for _, k := range deps {
			dep := Target{Table: ref.ownerTable(), Key: k}
			if err = p.add(dep, ref); err != nil {
				return err
			}
		}
	}
	return p.apply(ctx, tx, r)
}

// retargetHolders moves each holder in deps through m, through the reference that each
// names. It checks every reference before it rewrites a holder.
func (r *registry) retargetHolders(
	ctx context.Context,
	tx Tx,
	m moves,
	deps []Dependent,
	limit int,
) error {
	p := retargetPlan{m: m, limit: limit, indexOf: make(map[Target]int, len(deps))}
	for _, d := range deps {
		ref, ok := r.referenceTo(m.table, d.Holder.Table, d.Reference)
		if !ok {
			return errors.Newf(
				"[gorp] - table %s has no reference %s to table %s",
				d.Holder.Table,
				d.Reference,
				m.table,
			)
		}
		table, err := r.table(d.Holder.Table)
		if err != nil {
			return err
		}
		h := d.Holder
		if h.Key, err = table.formatKey(h.Key); err != nil {
			return err
		}
		if err = p.add(h, ref); err != nil {
			return err
		}
	}
	return p.apply(ctx, tx, r)
}

// moves is the map of one retarget between keys of one table, in the table's key type
// and as targets.
type moves struct {
	// table is the name of the table whose keys move.
	table string
	// keys is a map[K]K of the table's key type K.
	keys any
	// targets holds the same moves as keys. It is nil until withTargets fills it, which
	// a plan does once for a reference whose targets are not keys of the table.
	targets map[Target]Target
	// toTargets returns the targets of keys.
	toTargets func(table string, keys any) map[Target]Target
}

func newMoves[K Key](table string, to map[K]K) moves {
	return moves{table: table, keys: to, toTargets: movesTargets[K]}
}

func movesTargets[K Key](table string, keys any) map[Target]Target {
	to, ok := keys.(map[K]K)
	if !ok {
		panic("[gorp] - moves hold keys of another type than their table")
	}
	targets := make(map[Target]Target, len(to))
	for from, onto := range to {
		targets[NewTarget(table, from)] = NewTarget(table, onto)
	}
	return targets
}

// withTargets returns m with its targets filled.
func (m moves) withTargets() moves {
	if m.targets == nil {
		m.targets = m.toTargets(m.table, m.keys)
	}
	return m
}

// formatSources returns the keys of to as text, sorted, for an error.
func formatSources(to map[Target]Target) string {
	sources := make([]string, 0, len(to))
	for from := range to {
		sources = append(sources, from.Table+" "+from.Key)
	}
	slices.Sort(sources)
	return strings.Join(sources, ", ")
}
