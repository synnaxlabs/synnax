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
	"maps"
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
	// exists reports whether the entry with the text key k exists in snap.
	exists(ctx context.Context, snap Snapshot, k string) (bool, error)
	// issues reports whether the table has a TableConfig.Issued.
	issues() bool
	// issued reports whether the table issued the text key k.
	issued(k string) (bool, error)
	// deleteKeys deletes the entries with the text keys keys as a cascade.
	deleteKeys(ctx context.Context, tx Tx, keys []string) error
	// detach removes or inlines the targets of ops in the entry with the text key k and
	// writes it back as a detach.
	detach(ctx context.Context, tx Tx, k string, ops []detachOp) error
	// watches reports whether the table runs a hook when an entry of the table named
	// table that its entries hold is set.
	watches(table string) bool
	// targetSet runs the hook of the table for a set of an entry of the table named
	// table, with the entry of the text key k.
	targetSet(ctx context.Context, tx Tx, k, table string) error
}

// registeredReference is the part of a Reference that the delete walk reaches.
type registeredReference interface {
	ownerTable() string
	referenceName() string
	index() int
	onDeletePolicy() OnDelete
	linkKind() LinkKind
	referenceTag() uint8
	targetTables() []string
	// dependents returns, at each position of targets, the text keys of the entries
	// that hold that target, or nil when no entry holds any of targets. keys is nil or
	// a slice of the keys of targets, which a reference whose values have that type
	// reads instead of parsing targets.
	dependents(
		ctx context.Context,
		snap Snapshot,
		targets []Target,
		keys any,
	) ([][]string, error)
	// dependentsOf returns the text keys of the entries that hold target.
	dependentsOf(ctx context.Context, snap Snapshot, target Target) ([]string, error)
}

// detachOp removes or inlines target in the reference at index ref of an entry.
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
// cannot share a name or a prefix, and a table that a Weak reference points at must
// have a TableConfig.Issued.
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
	for _, ref := range r.referencing[name] {
		if err := checkIssues(ref, name, table); err != nil {
			return err
		}
	}
	r.tables[name] = table
	r.names[string(prefix)] = name
	return nil
}

// referenced returns the names of the tables that a reference may point at, in
// ascending order.
func (r *registry) referenced() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return slices.Sorted(maps.Keys(r.referencing))
}

// tableName returns the name of the registered table that stores under prefix.
func (r *registry) tableName(prefix []byte) (string, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	name, ok := r.names[string(prefix)]
	return name, ok
}

// addReferences adds refs to the references that deletes of their target tables
// follow. It fails, adding none, when a Weak reference points at an open table with no
// TableConfig.Issued.
func (r *registry) addReferences(refs []registeredReference) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, ref := range refs {
		for _, target := range ref.targetTables() {
			if t, ok := r.tables[target]; ok {
				if err := checkIssues(ref, target, t); err != nil {
					return err
				}
			}
		}
	}
	for _, ref := range refs {
		for _, target := range ref.targetTables() {
			r.referencing[target] = append(r.referencing[target], ref)
		}
	}
	return nil
}

// checkIssues fails when ref is Weak and its target table, named name, has no
// TableConfig.Issued.
func checkIssues(ref registeredReference, name string, table registeredTable) error {
	if ref.onDeletePolicy() != Weak || table.issues() {
		return nil
	}
	return errors.Newf(
		"[gorp] - weak reference %s of table %q points at table %q, which has no Issued",
		ref.referenceName(),
		ref.ownerTable(),
		name,
	)
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

// dependents returns, at each position of targets, the entries that hold that target
// through any registered reference. Every target is an entry of the table named table,
// and keys is a slice of their keys, in order.
func (r *registry) dependents(
	ctx context.Context,
	snap Snapshot,
	table string,
	targets []Target,
	keys any,
) ([][]Dependent, error) {
	type found struct {
		ref     registeredReference
		holders [][]string
	}
	var (
		all    []found
		counts = make([]int, len(targets))
	)
	for _, ref := range r.referencesTo(table) {
		holders, err := ref.dependents(ctx, snap, targets, keys)
		if err != nil {
			return nil, err
		}
		if holders == nil {
			continue
		}
		all = append(all, found{ref: ref, holders: holders})
		for i, held := range holders {
			counts[i] += len(held)
		}
	}
	out := make([][]Dependent, len(targets))
	for i, n := range counts {
		if n > 0 {
			out[i] = make([]Dependent, 0, n)
		}
	}
	for _, f := range all {
		for i, held := range f.holders {
			for _, k := range held {
				out[i] = append(out[i], Dependent{
					Holder:    Target{Table: f.ref.ownerTable(), Key: k},
					Reference: f.ref.referenceName(),
					Kind:      f.ref.linkKind(),
					Tag:       f.ref.referenceTag(),
				})
			}
		}
	}
	return out, nil
}

// watchers returns the Use references to the table named table whose owner tables
// watch it.
func (r *registry) watchers(table string) []registeredReference {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []registeredReference
	for _, ref := range r.referencing[table] {
		if ref.linkKind() != Use {
			continue
		}
		if t, ok := r.tables[ref.ownerTable()]; ok && t.watches(table) {
			out = append(out, ref)
		}
	}
	return out
}

// targetSet runs the target hook of each entry that holds target through one of refs,
// which watchers returns for the table of target.
func (r *registry) targetSet(
	ctx context.Context,
	tx Tx,
	refs []registeredReference,
	target Target,
) error {
	for _, ref := range refs {
		owner, err := r.table(ref.ownerTable())
		if err != nil {
			return err
		}
		keys, err := ref.dependentsOf(ctx, tx, target)
		if err != nil {
			return err
		}
		for _, k := range keys {
			if err = owner.targetSet(ctx, tx, k, target.Table); err != nil {
				return err
			}
		}
	}
	return nil
}

// deletePlan is the full effect of deleting a set of entries.
type deletePlan struct {
	// table is the name of the table of the entries the caller asked to delete.
	table string
	// requested holds the text keys of the entries the caller asked to delete.
	requested set.Set[string]
	// cascades holds the text keys of the other entries to delete, by table name.
	cascades map[string]set.Set[string]
	// detaches holds the targets to remove or inline in each entry the delete keeps. An
	// entry leaves detaches when the plan adds it to cascades.
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
// the referencing entry too, and Detach and Inline remove or replace the target in a
// referencing entry the delete keeps. plan reads only; it changes nothing. It fails
// when the delete would delete or rewrite more than limit entries.
func (r *registry) plan(
	ctx context.Context,
	tx Tx,
	table string,
	keys []string,
	limit int,
) (*deletePlan, error) {
	w := newPlanWalk(table, keys, limit)
	if w.plan.changes() > limit {
		return nil, w.tooLarge()
	}
	for len(w.queue) > 0 {
		target := w.queue[0]
		w.queue = w.queue[1:]
		for _, ref := range r.referencesTo(target.Table) {
			deps, err := ref.dependentsOf(ctx, tx, target)
			if err != nil {
				return nil, err
			}
			for _, k := range deps {
				dep := Target{Table: ref.ownerTable(), Key: k}
				if err := w.visit(ref, dep, target); err != nil {
					return nil, err
				}
			}
		}
	}
	if err := w.checkRestrictions(); err != nil {
		return nil, err
	}
	return w.plan, nil
}

// restriction is a Restrict reference through which dependent holds target.
type restriction struct {
	ref       registeredReference
	dependent Target
	target    Target
}

func (rs restriction) err() error {
	return RestrictError{
		Target:    rs.target,
		Holder:    rs.dependent,
		Reference: rs.ref.referenceName(),
	}
}

// planWalk is the state of a breadth-first walk that builds a deletePlan.
type planWalk struct {
	plan         *deletePlan
	keys         []string
	limit        int
	queue        []Target
	restrictions []restriction
	restricted   set.Set[Target]
}

func newPlanWalk(table string, keys []string, limit int) *planWalk {
	w := &planWalk{
		plan: &deletePlan{
			table:     table,
			requested: set.New[string](),
			cascades:  make(map[string]set.Set[string]),
			detaches:  make(map[Target][]detachOp),
		},
		keys:       keys,
		limit:      limit,
		restricted: set.New[Target](),
	}
	for _, k := range keys {
		if w.plan.request(k) {
			w.queue = append(w.queue, Target{Table: table, Key: k})
		}
	}
	return w
}

func (w *planWalk) tooLarge() error {
	named := w.plan.table + " " + w.keys[0]
	if len(w.keys) > 1 {
		named = strconv.Itoa(len(w.keys)) + " entries of " + w.plan.table
	}
	return errors.Wrapf(
		validate.ErrValidation,
		"deleting %s would change more than %d entries",
		named,
		w.limit,
	)
}

// visit applies the delete policy of ref to dep, an entry that holds target through
// ref.
func (w *planWalk) visit(ref registeredReference, dep, target Target) error {
	switch ref.onDeletePolicy() {
	case Cascade:
		if !w.plan.cascade(dep) {
			return nil
		}
		w.queue = append(w.queue, dep)
	case Restrict:
		if w.plan.contains(dep) || w.restricted.Contains(dep) {
			return nil
		}
		w.restricted.Add(dep)
		w.restrictions = append(
			w.restrictions,
			restriction{ref: ref, dependent: dep, target: target},
		)
		// The closure holds at most limit entries, so at least one restricted
		// dependent stays outside it.
		if len(w.restrictions) > w.limit {
			if err := w.checkRestrictions(); err != nil {
				return err
			}
		}
	case Weak:
		return nil
	case Detach, Inline:
		if w.plan.contains(dep) {
			return nil
		}
		w.plan.detaches[dep] = append(
			w.plan.detaches[dep],
			detachOp{ref: ref.index(), target: target},
		)
	}
	if w.plan.changes() > w.limit {
		return w.tooLarge()
	}
	return nil
}

// checkRestrictions returns a RestrictError for the first restricted dependent that
// the plan does not delete.
func (w *planWalk) checkRestrictions() error {
	for _, rs := range w.restrictions {
		if !w.plan.contains(rs.dependent) {
			return rs.err()
		}
	}
	return nil
}

// apply carries out what p adds to the delete that the caller asked for: it detaches
// first, then cascades, so each Inline reads a target that still exists. It leaves the
// entries the caller asked to delete in place.
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
