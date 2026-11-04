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
	"slices"

	"github.com/synnaxlabs/gorp/internal/index"
	"github.com/synnaxlabs/gorp/internal/key"
	"github.com/synnaxlabs/x/errors"
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

// Target identifies the entry that a reference points at.
type Target struct {
	// Table is the name of the target's table.
	Table string
	// Key is the target's primary key in the text form that FormatKey returns.
	Key string
}

// To returns the Target of the entry with the given key in the table named table.
func To[K Key](
	table string,
	k K,
) Target {
	return Target{Table: table, Key: FormatKey(k)}
}

// FormatKey returns the text form of k: the string itself for a string key, base 10
// for an integer key, and the 8-4-4-4-12 lowercase hex form for a 16-byte key. A
// Target holds a key in this form only.
func FormatKey[K Key](k K) string { return key.Format(k) }

// ReferenceConfig configures a Reference.
type ReferenceConfig[K Key, E Entry[K]] struct {
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

// Reference is a field of E that holds the keys of other entries. Register it on a
// Table through TableConfig.References. Gorp then checks on every write that each
// target exists, keeps an index from each target to the entries that hold it, and
// applies OnDelete when a target is deleted. All of it runs in the caller's
// transaction.
//
// Integrity holds within one node only. Gorp does not isolate concurrent transactions,
// so a check and a conflicting write in another transaction can both commit.
type Reference[K Key, E Entry[K]] struct {
	cfg   ReferenceConfig[K, E]
	store *index.Multi[*txState, K, Target]
	// owner is the name of the table the reference is registered on.
	owner string
	// position is the index of the reference in its table's references.
	position int
}

// NewReference returns a Reference from cfg. OpenTable validates cfg.
func NewReference[K Key, E Entry[K]](cfg ReferenceConfig[K, E]) *Reference[K, E] {
	return &Reference[K, E]{cfg: cfg, store: index.NewMulti[*txState, K, Target]()}
}

func (r *Reference[K, E]) validate(table string) error {
	v := validate.New("gorp.reference")
	v.NotEmptyString("name", r.cfg.Name)
	v.NotEmptySlice("tables", r.cfg.Tables)
	v.NotNil("targets", r.cfg.Targets)
	v.Ternary(
		"on_delete",
		r.cfg.OnDelete < Restrict || r.cfg.OnDelete > Detach,
		"must be Restrict, Cascade, or Detach",
	)
	v.Ternary(
		"detach",
		r.cfg.OnDelete == Detach && r.cfg.Detach == nil,
		"must be set when on_delete is Detach",
	)
	v.Ternary(
		"acyclic",
		r.cfg.Acyclic && !slices.Contains(r.cfg.Tables, table),
		"requires the reference to point at its own table",
	)
	return v.Error()
}

// Name implements Index.
func (r *Reference[K, E]) Name() string { return r.cfg.Name }

//nolint:unused
func (r *Reference[K, E]) populate() (func(E), func(error)) {
	insert, finish := r.store.Populate()
	return func(entry E) { insert(entry.GorpKey(), r.cfg.Targets(&entry)) },
		func(err error) { finish(populateErr(r.cfg.Name, err)) }
}

//nolint:unused
func (r *Reference[K, E]) set(entry E) {
	r.store.Set(entry.GorpKey(), r.cfg.Targets(&entry))
}

//nolint:unused
func (r *Reference[K, E]) delete(key K) { r.store.Delete(key) }

//nolint:unused
func (r *Reference[K, E]) stageSet(tx Tx, entry E) {
	r.store.Stage(tx.txIdentity(), entry.GorpKey(), r.cfg.Targets(&entry))
}

//nolint:unused
func (r *Reference[K, E]) stageDelete(tx Tx, key K) {
	r.store.StageDelete(tx.txIdentity(), key)
}

//nolint:unused
func (r *Reference[K, E]) announce(entry *E) uint64 {
	return r.store.Announce((*entry).GorpKey(), r.cfg.Targets(entry))
}

//nolint:unused
func (r *Reference[K, E]) withdraw(handle uint64) { r.store.Withdraw(handle) }

// Get returns the keys of the entries that hold any of targets, with the changes
// staged against tx applied. Pass a nil tx to read committed state only. Get blocks
// until the index has finished populating, and returns ErrIndexInvalid if population
// failed.
func (r *Reference[K, E]) Get(tx Tx, targets ...Target) ([]K, error) {
	return r.store.Get(scopeOf(tx), targets...)
}

// Filter returns a Filter matching entries that hold any of targets. It sees the
// changes staged in the transaction it runs in.
func (r *Reference[K, E]) Filter(targets ...Target) Filter[K, E] {
	return indexFilter[K, E](r.store, func(e *E, values []Target) bool {
		return slices.ContainsFunc(r.cfg.Targets(e), func(t Target) bool {
			return slices.Contains(values, t)
		})
	}, targets)
}

// dependents implements registeredReference.
func (r *Reference[K, E]) dependents(tx Tx, target Target) ([]string, error) {
	keys, err := r.store.Get(scopeOf(tx), target)
	if err != nil {
		return nil, err
	}
	out := make([]string, len(keys))
	for i, k := range keys {
		out[i] = FormatKey(k)
	}
	return out, nil
}

func (r *Reference[K, E]) ownerTable() string    { return r.owner }
func (r *Reference[K, E]) referenceName() string { return r.cfg.Name }
func (r *Reference[K, E]) index() int            { return r.position }
func (r *Reference[K, E]) onDelete() OnDelete    { return r.cfg.OnDelete }
func (r *Reference[K, E]) tables() []string      { return r.cfg.Tables }

// fieldErr returns a validation error on the reference's field.
func (r *Reference[K, E]) fieldErr(format string, args ...any) error {
	return validate.PathedError(
		errors.Wrapf(validate.ErrValidation, format, args...),
		r.cfg.Name,
	)
}
