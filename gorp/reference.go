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

	"github.com/synnaxlabs/gorp/internal/index"
	"github.com/synnaxlabs/gorp/internal/key"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/validate"
)

// OnDelete is what Gorp does to an entry when an entry it references is deleted.
type OnDelete uint8

const (
	// Restrict fails the delete while the referencing entry exists and the delete does
	// not remove it too.
	Restrict OnDelete = iota + 1
	// Cascade deletes the referencing entry along with its target.
	Cascade
	// Detach keeps the referencing entry and removes the deleted target from it.
	Detach
)

// Target identifies an entry by the name of its table and its key in the text form
// that FormatKey returns.
type Target struct {
	// Table is the name of the target's table.
	Table string
	// Key is the target's primary key in the text form that FormatKey returns.
	Key string
}

// NewTarget returns the Target of the entry with key k in the table named table.
func NewTarget[K Key](table string, k K) Target {
	return Target{Table: table, Key: FormatKey(k)}
}

// FormatKey returns the text form of k: the string itself for a string key, base 10
// for an integer key, and the 8-4-4-4-12 lowercase hex form for a 16-byte key. A
// Target holds a key in this form only.
func FormatKey[K Key](k K) string { return key.Format(k) }

// Reference is a field of E that holds the keys of other entries. Register it on a
// Table through TableConfig.References. Gorp then checks on every write that each
// target exists, keeps an index from each target to the entries that hold it, and
// applies its OnDelete when a target is deleted. All of it runs in the caller's
// transaction.
//
// KeyReference and TargetReference implement Reference. Integrity holds within one
// node only.
type Reference[K Key, E Entry[K]] interface {
	Index[K, E]
	// validate checks the configuration of the reference for a table named table.
	validate(table string) error
	// bind registers the reference at position in the references of the table named
	// owner.
	bind(owner string, position int)
	// targets returns the targets that e holds.
	targets(e *E) []Target
	// detachTarget removes target from e. It reports false when the reference has no
	// Detach.
	detachTarget(e *E, target Target) bool
	// check validates the targets that e holds against tx, for a write of e to t.
	check(ctx context.Context, tx Tx, t *Table[K, E], e *E) error
	registeredReference
}

// referenceSpec is the part of a reference that does not depend on its target type.
type referenceSpec struct {
	name     string
	tables   []string
	onDelete OnDelete
	unique   bool
	acyclic  bool
	// textKeys is true when a target names its key as caller-provided text, which a
	// write must check against the text form of its table's keys.
	textKeys bool
	// detachable is true when the reference can remove a target from an entry.
	detachable bool
	// tablesField names the config field that holds tables, for validation errors.
	tablesField string
	// owner is the name of the table the reference is registered on.
	owner string
	// position is the index of the reference in its table's references.
	position int
}

func (s *referenceSpec) validate(table string, targets bool) error {
	v := validate.New("gorp.reference")
	v.NotEmptyString("name", s.name)
	v.Ternary(
		s.tablesField,
		len(s.tables) == 0 || slices.Contains(s.tables, ""),
		"must name at least one table",
	)
	v.Ternary("targets", !targets, "must be non-nil")
	v.Ternary(
		"on_delete",
		s.onDelete < Restrict || s.onDelete > Detach,
		"must be Restrict, Cascade, or Detach",
	)
	v.Ternary(
		"detach",
		s.onDelete == Detach && !s.detachable,
		"must be set when on_delete is Detach",
	)
	v.Ternary(
		"acyclic",
		s.acyclic && !slices.Contains(s.tables, table),
		"requires the reference to point at its own table",
	)
	return v.Error()
}

// fieldErr returns a validation error on the reference's field.
func (s *referenceSpec) fieldErr(format string, args ...any) error {
	return validate.PathedError(
		errors.Wrapf(validate.ErrValidation, format, args...),
		s.name,
	)
}

// KeyReferenceConfig configures a KeyReference.
type KeyReferenceConfig[K Key, E Entry[K], T Key] struct {
	// Name names the reference in errors. Use the name of the field that holds it.
	//
	// [REQUIRED]
	Name string
	// Table is the name of the table that holds the targets.
	//
	// [REQUIRED]
	Table string
	// Targets returns the keys of the targets that e holds, or nil when it holds none.
	//
	// [REQUIRED]
	Targets func(e *E) []T
	// OnDelete is what happens to e when one of its targets is deleted.
	//
	// [REQUIRED]
	OnDelete OnDelete
	// Detach removes the target with key target from e.
	//
	// [REQUIRED] when OnDelete is Detach.
	Detach func(e *E, target T)
	// Unique allows at most one entry to hold each target.
	Unique bool
	// Acyclic rejects a write that makes a chain of references within the table return
	// to its start. Table must be the table the reference is registered on.
	Acyclic bool
}

// KeyReference is a Reference to the entries of one table, which holds each target as
// a key of type T. Construct one with NewKeyReference.
type KeyReference[K Key, E Entry[K], T Key] struct {
	reference[K, E, T]
}

// NewKeyReference returns a KeyReference from cfg. OpenTable validates cfg.
func NewKeyReference[K Key, E Entry[K], T Key](
	cfg KeyReferenceConfig[K, E, T],
) *KeyReference[K, E, T] {
	table := cfg.Table
	return &KeyReference[K, E, T]{
		name:        cfg.Name,
		tables:      []string{table},
		tablesField: "table",
		onDelete:    cfg.OnDelete,
		unique:      cfg.Unique,
		acyclic:     cfg.Acyclic,
		detachable:  cfg.Detach != nil,
		targetsOf:   cfg.Targets,
		detach:      cfg.Detach,
		format:      func(t T) Target { return Target{Table: table, Key: key.Format(t)} },
		parse: func(target Target) (T, bool) {
			if target.Table != table {
				var zero T
				return zero, false
			}
			t, err := key.Parse[T](target.Key)
			return t, err == nil
		},
		store: index.NewMulti[*txState, K, T](),
	}
}

// TargetReferenceConfig configures a TargetReference.
type TargetReferenceConfig[K Key, E Entry[K]] struct {
	// Name names the reference in errors. Use the name of the field that holds it.
	//
	// [REQUIRED]
	Name string
	// Tables are the names of the tables the reference may point at. A write whose
	// target is in any other table fails validation.
	//
	// [REQUIRED]
	Tables []string
	// Targets returns the targets that e holds, or nil when it holds none.
	//
	// [REQUIRED]
	Targets func(e *E) []Target
	// OnDelete is what happens to e when one of its targets is deleted.
	//
	// [REQUIRED]
	OnDelete OnDelete
	// Detach removes target from e.
	//
	// [REQUIRED] when OnDelete is Detach.
	Detach func(e *E, target Target)
	// Unique allows at most one entry to hold each target.
	Unique bool
	// Acyclic rejects a write that makes a chain of references within the table return
	// to its start. The table must be one of Tables.
	Acyclic bool
}

// TargetReference is a Reference to the entries of any of several tables, which holds
// each target as a Target. A write fails validation when a target's key is not in the
// text form of its table's keys. Construct one with NewTargetReference.
type TargetReference[K Key, E Entry[K]] struct {
	reference[K, E, Target]
}

// NewTargetReference returns a TargetReference from cfg. OpenTable validates cfg.
func NewTargetReference[K Key, E Entry[K]](
	cfg TargetReferenceConfig[K, E],
) *TargetReference[K, E] {
	return &TargetReference[K, E]{
		name:        cfg.Name,
		tables:      cfg.Tables,
		tablesField: "tables",
		onDelete:    cfg.OnDelete,
		unique:      cfg.Unique,
		acyclic:     cfg.Acyclic,
		textKeys:    true,
		detachable:  cfg.Detach != nil,
		targetsOf:   cfg.Targets,
		detach:      cfg.Detach,
		format:      func(t Target) Target { return t },
		parse:       func(t Target) (Target, bool) { return t, true },
		store:       index.NewMulti[*txState, K, Target](),
	}
}

// reference is the engine of a Reference whose targets are values of type T.
type reference[K Key, E Entry[K], T comparable] struct {
	referenceSpec
	targetsOf func(e *E) []T
	detach    func(e *E, target T)
	// format returns the Target of t.
	format func(t T) Target
	// parse returns the value of target, or false when the reference cannot hold it.
	parse func(target Target) (T, bool)
	store *index.Multi[*txState, K, T]
}

// Name implements Index.
func (r *reference[K, E, T]) Name() string { return r.name }

func (r *reference[K, E, T]) validate(table string) error {
	return r.referenceSpec.validate(table, r.targetsOf != nil)
}

func (r *reference[K, E, T]) bind(owner string, position int) {
	r.owner, r.position = owner, position
}

//nolint:unused
func (r *reference[K, E, T]) populate() (func(E), func(error)) {
	insert, finish := r.store.Populate()
	return func(entry E) { insert(entry.GorpKey(), r.targetsOf(&entry)) },
		func(err error) { finish(populateErr(r.name, err)) }
}

//nolint:unused
func (r *reference[K, E, T]) set(entry E) {
	r.store.Set(entry.GorpKey(), r.targetsOf(&entry))
}

//nolint:unused
func (r *reference[K, E, T]) delete(key K) { r.store.Delete(key) }

//nolint:unused
func (r *reference[K, E, T]) stageSet(tx Tx, entry E) {
	r.store.Stage(tx.txIdentity(), entry.GorpKey(), r.targetsOf(&entry))
}

//nolint:unused
func (r *reference[K, E, T]) stageDelete(tx Tx, key K) {
	r.store.StageDelete(tx.txIdentity(), key)
}

func (r *reference[K, E, T]) targets(e *E) []Target {
	held := r.targetsOf(e)
	if len(held) == 0 {
		return nil
	}
	out := make([]Target, len(held))
	for i, t := range held {
		out[i] = r.format(t)
	}
	return out
}

func (r *reference[K, E, T]) detachTarget(e *E, target Target) bool {
	if r.detach == nil {
		return false
	}
	if t, ok := r.parse(target); ok {
		r.detach(e, t)
	}
	return true
}

// Get returns the keys of the entries that hold any of targets, with the changes
// staged against tx applied. Pass a nil tx to skip staged changes. See LookupIndex.Get
// for its contract.
func (r *reference[K, E, T]) Get(
	ctx context.Context,
	tx Tx,
	targets ...T,
) ([]K, error) {
	return probe(ctx, r.store, tx, targets)
}

// Filter returns a Filter matching entries that hold any of targets. It sees the
// changes staged in the transaction it runs in.
func (r *reference[K, E, T]) Filter(targets ...T) Filter[K, E] {
	return indexFilter[K, E](r.store, func(e *E, values []T) bool {
		return slices.ContainsFunc(r.targetsOf(e), func(t T) bool {
			return slices.Contains(values, t)
		})
	}, targets)
}

// dependents implements registeredReference.
func (r *reference[K, E, T]) dependents(
	ctx context.Context,
	tx Tx,
	target Target,
) ([]string, error) {
	t, ok := r.parse(target)
	if !ok {
		return nil, nil
	}
	keys, err := probe(ctx, r.store, tx, []T{t})
	if err != nil {
		return nil, err
	}
	out := make([]string, len(keys))
	for i, k := range keys {
		out[i] = FormatKey(k)
	}
	return out, nil
}

func (r *reference[K, E, T]) ownerTable() string       { return r.owner }
func (r *reference[K, E, T]) referenceName() string    { return r.name }
func (r *reference[K, E, T]) index() int               { return r.position }
func (r *reference[K, E, T]) onDeletePolicy() OnDelete { return r.onDelete }
func (r *reference[K, E, T]) targetTables() []string   { return r.tables }

// check implements Reference.
func (r *reference[K, E, T]) check(
	ctx context.Context,
	tx Tx,
	t *Table[K, E],
	e *E,
) error {
	held := r.targetsOf(e)
	for _, v := range held {
		target := r.format(v)
		if r.textKeys && !slices.Contains(r.tables, target.Table) {
			return r.fieldErr(
				"%s cannot reference %s %s",
				t.name,
				target.Table,
				target.Key,
			)
		}
		table, err := t.db.registry.table(target.Table)
		if err != nil {
			return err
		}
		if r.textKeys && !table.canonical(target.Key) {
			return r.fieldErr("%q is not a key of %s", target.Key, target.Table)
		}
		ok, err := table.exists(ctx, tx, target.Key)
		if err != nil {
			return err
		}
		if !ok {
			return r.fieldErr("%s %s does not exist", target.Table, target.Key)
		}
	}
	if r.unique {
		if err := r.checkUnique(ctx, tx, t.name, e, held); err != nil {
			return err
		}
	}
	if r.acyclic {
		return r.checkAcyclic(ctx, tx, t, e)
	}
	return nil
}

func (r *reference[K, E, T]) checkUnique(
	ctx context.Context,
	tx Tx,
	table string,
	e *E,
	held []T,
) error {
	self := (*e).GorpKey()
	for _, v := range held {
		holders, err := probe(ctx, r.store, tx, []T{v})
		if err != nil {
			return err
		}
		for _, h := range holders {
			if h != self {
				target := r.format(v)
				return r.fieldErr(
					"%s %s already references %s %s",
					table,
					FormatKey(h),
					target.Table,
					target.Key,
				)
			}
		}
	}
	return nil
}

// checkAcyclic walks the chain of targets in t from e and fails if the chain reaches
// e.
func (r *reference[K, E, T]) checkAcyclic(
	ctx context.Context,
	tx Tx,
	t *Table[K, E],
	e *E,
) error {
	self := Target{Table: t.name, Key: FormatKey((*e).GorpKey())}
	visited := set.New[Target]()
	frontier := r.targets(e)
	for len(frontier) > 0 {
		next := frontier[len(frontier)-1]
		frontier = frontier[:len(frontier)-1]
		if visited.Contains(next) || next.Table != t.name {
			continue
		}
		if next == self {
			return r.fieldErr("%s %s would form a cycle", t.name, self.Key)
		}
		visited.Add(next)
		pk, err := key.Parse[K](next.Key)
		if err != nil {
			return err
		}
		var parent E
		err = t.NewRetrieve().Where(MatchKeys[K, E](pk)).Entry(&parent).Exec(ctx, tx)
		if errors.Is(err, query.ErrNotFound) {
			continue
		}
		if err != nil {
			return err
		}
		frontier = append(frontier, r.targets(&parent)...)
	}
	return nil
}
