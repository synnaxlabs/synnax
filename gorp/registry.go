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
	"sync"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/validate"
)

// registeredTable is the part of a Table that references in other tables reach.
type registeredTable interface {
	// canonical reports whether k is a key of the table in the text form that
	// FormatKey returns.
	canonical(k string) bool
	// exists reports whether the entry with the text key k exists in tx.
	exists(ctx context.Context, tx Tx, k string) (bool, error)
	// deleteKeys deletes the entries with the text keys keys as a cascade.
	deleteKeys(ctx context.Context, tx Tx, keys []string) error
	// detach removes the targets of ops from the entry with the text key k and writes
	// it back as a detach.
	detach(ctx context.Context, tx Tx, k string, ops []detachOp) error
}

// registeredReference is the part of a Reference that the delete walk reaches.
type registeredReference interface {
	ownerTable() string
	referenceName() string
	index() int
	onDeletePolicy() OnDelete
	targetTables() []string
	// dependents returns the text keys of the entries that hold target.
	dependents(ctx context.Context, snap Snapshot, target Target) ([]string, error)
}

// detachOp removes target from the reference at index ref of an entry.
type detachOp struct {
	ref    int
	target Target
}

// registry holds the tables of a DB that take part in references, by name, and the
// open tables that have indexes, by key prefix.
type registry struct {
	// mu guards tables, names, referencing, and indexed.
	mu     sync.RWMutex
	tables map[string]registeredTable
	// names maps a key prefix to the name of the registered table that stores under it.
	names map[string]string
	// referencing maps a table name to the references that may point at it.
	referencing map[string][]registeredReference
	// indexed maps a key prefix to the open tables with indexes that store under it.
	indexed map[string][]any
}

func newRegistry() *registry {
	return &registry{
		tables:      make(map[string]registeredTable),
		names:       make(map[string]string),
		referencing: make(map[string][]registeredReference),
		indexed:     make(map[string][]any),
	}
}

// addIndexed adds table, which stores under prefix, to the tables that writers keep
// the indexes of.
func (r *registry) addIndexed(prefix []byte, table any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.indexed[string(prefix)] = append(r.indexed[string(prefix)], table)
}

// removeIndexed undoes addIndexed.
func (r *registry) removeIndexed(prefix []byte, table any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	kept := slices.DeleteFunc(
		slices.Clone(r.indexed[string(prefix)]),
		func(t any) bool { return t == table },
	)
	if len(kept) == 0 {
		delete(r.indexed, string(prefix))
		return
	}
	r.indexed[string(prefix)] = kept
}

// indexedTables returns the open tables with indexes that store under prefix.
func (r *registry) indexedTables(prefix []byte) []any {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return slices.Clone(r.indexed[string(prefix)])
}

// register adds table, named name, which stores under prefix. Two registered tables
// cannot share a name or a prefix.
func (r *registry) register(name string, prefix []byte, table registeredTable) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.tables[name]; ok {
		return errors.Newf("[gorp] - table %q is already open", name)
	}
	if other, ok := r.names[string(prefix)]; ok {
		return errors.Newf(
			"[gorp] - table %q stores under the key prefix of open table %q",
			name,
			other,
		)
	}
	r.tables[name] = table
	r.names[string(prefix)] = name
	return nil
}

// tableName returns the name of the registered table that stores under prefix.
func (r *registry) tableName(prefix []byte) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	name, ok := r.names[string(prefix)]
	return name, ok
}

// addReferences adds refs to the references that deletes of their target tables
// follow.
func (r *registry) addReferences(refs []registeredReference) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, ref := range refs {
		for _, target := range ref.targetTables() {
			r.referencing[target] = append(r.referencing[target], ref)
		}
	}
}

func (r *registry) unregister(name string, prefix []byte) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.tables, name)
	delete(r.names, string(prefix))
	for target, refs := range r.referencing {
		kept := refs[:0]
		for _, ref := range refs {
			if ref.ownerTable() != name {
				kept = append(kept, ref)
			}
		}
		if len(kept) == 0 {
			delete(r.referencing, target)
			continue
		}
		r.referencing[target] = kept
	}
}

func (r *registry) table(name string) (registeredTable, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.tables[name]
	if !ok {
		return nil, errors.Newf("[gorp] - no open table is named %q", name)
	}
	return t, nil
}

// referencesTo returns the references that may point at the table named table.
func (r *registry) referencesTo(table string) []registeredReference {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return slices.Clone(r.referencing[table])
}

// dependents returns the entries that hold target through any registered reference.
func (r *registry) dependents(
	ctx context.Context,
	snap Snapshot,
	target Target,
) ([]Target, error) {
	var out []Target
	for _, ref := range r.referencesTo(target.Table) {
		keys, err := ref.dependents(ctx, snap, target)
		if err != nil {
			return nil, err
		}
		for _, k := range keys {
			out = append(out, Target{Table: ref.ownerTable(), Key: k})
		}
	}
	return out, nil
}

// deletePlan is the full effect of deleting a set of entries.
type deletePlan struct {
	// table is the name of the table of the entries the caller asked to delete.
	table string
	// requested holds the text keys of the entries the caller asked to delete.
	requested set.Set[string]
	// cascades holds the text keys of the other entries to delete, by table name.
	cascades map[string]set.Set[string]
	// detaches holds the targets to remove from each entry the delete keeps. An entry
	// leaves detaches when the plan adds it to cascades.
	detaches map[Target][]detachOp
	// size is the number of entries the plan deletes.
	size int
}

// changes returns the number of entries the plan deletes or rewrites.
func (p *deletePlan) changes() int { return p.size + len(p.detaches) }

func (p *deletePlan) contains(t Target) bool {
	if t.Table == p.table && p.requested.Contains(t.Key) {
		return true
	}
	keys, ok := p.cascades[t.Table]
	return ok && keys.Contains(t.Key)
}

// request adds the entry with the text key k to the entries the caller asked to delete.
// It reports false when the plan already deletes the entry.
func (p *deletePlan) request(k string) bool {
	if p.requested.Contains(k) {
		return false
	}
	p.requested.Add(k)
	p.size++
	return true
}

// cascade adds t to the entries that a cascade deletes. It reports false when the plan
// already deletes t.
func (p *deletePlan) cascade(t Target) bool {
	if p.contains(t) {
		return false
	}
	keys, ok := p.cascades[t.Table]
	if !ok {
		keys = set.New[string]()
		p.cascades[t.Table] = keys
	}
	keys.Add(t.Key)
	p.size++
	delete(p.detaches, t)
	return true
}

// plan returns the effect of deleting the entries with the text keys keys from the
// table named table. It walks the references that point at each deleted entry: Cascade
// adds the referencing entry to the delete, Restrict fails unless the delete removes
// the referencing entry too, and Detach removes the target from a referencing entry
// the delete keeps. plan reads only; it changes nothing. It fails when the delete would
// delete or rewrite more than limit entries.
func (r *registry) plan(
	ctx context.Context,
	tx Tx,
	table string,
	keys []string,
	limit int,
) (*deletePlan, error) {
	p := &deletePlan{
		table:     table,
		requested: set.New[string](),
		cascades:  make(map[string]set.Set[string]),
		detaches:  make(map[Target][]detachOp),
	}
	tooLarge := func() error {
		named := table + " " + keys[0]
		if len(keys) > 1 {
			named = strconv.Itoa(len(keys)) + " entries of " + table
		}
		return errors.Wrapf(
			validate.ErrValidation,
			"deleting %s would change more than %d entries",
			named,
			limit,
		)
	}
	var queue []Target
	for _, k := range keys {
		if p.request(k) {
			queue = append(queue, Target{Table: table, Key: k})
		}
	}
	if p.changes() > limit {
		return nil, tooLarge()
	}
	type restriction struct {
		ref       registeredReference
		dependent Target
		target    Target
	}
	var (
		restrictions []restriction
		restricted   = set.New[Target]()
	)
	refuse := func(rs restriction) error {
		return RestrictError{
			Target:    rs.target,
			Holder:    rs.dependent,
			Reference: rs.ref.referenceName(),
		}
	}
	for len(queue) > 0 {
		target := queue[0]
		queue = queue[1:]
		for _, ref := range r.referencesTo(target.Table) {
			deps, err := ref.dependents(ctx, tx, target)
			if err != nil {
				return nil, err
			}
			for _, k := range deps {
				dep := Target{Table: ref.ownerTable(), Key: k}
				switch ref.onDeletePolicy() {
				case Cascade:
					if !p.cascade(dep) {
						continue
					}
					queue = append(queue, dep)
				case Restrict:
					if p.contains(dep) || restricted.Contains(dep) {
						continue
					}
					restricted.Add(dep)
					restrictions = append(
						restrictions,
						restriction{ref: ref, dependent: dep, target: target},
					)
					// The closure holds at most limit entries, so at least one
					// restricted dependent stays outside it.
					if len(restrictions) > limit {
						for _, rs := range restrictions {
							if !p.contains(rs.dependent) {
								return nil, refuse(rs)
							}
						}
					}
				case Detach:
					if p.contains(dep) {
						continue
					}
					p.detaches[dep] = append(
						p.detaches[dep],
						detachOp{ref: ref.index(), target: target},
					)
				}
				if p.changes() > limit {
					return nil, tooLarge()
				}
			}
		}
	}
	for _, rs := range restrictions {
		if !p.contains(rs.dependent) {
			return nil, refuse(rs)
		}
	}
	return p, nil
}

// apply carries out what p adds to the delete that the caller asked for: it detaches
// first, then cascades. It leaves the entries the caller asked to delete in place.
func (r *registry) apply(ctx context.Context, tx Tx, p *deletePlan) error {
	for dep, ops := range p.detaches {
		t, err := r.table(dep.Table)
		if err != nil {
			return err
		}
		if err := t.detach(ctx, tx, dep.Key, ops); err != nil {
			return err
		}
	}
	for name, keys := range p.cascades {
		t, err := r.table(name)
		if err != nil {
			return err
		}
		if err := t.deleteKeys(ctx, tx, keys.Slice()); err != nil {
			return err
		}
	}
	return nil
}
