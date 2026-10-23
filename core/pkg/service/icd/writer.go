// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package icd

import (
	"context"
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/service/library"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/validate"
)

// Writer creates, updates, and deletes ICDs within a transaction.
type Writer struct {
	tx        gorp.Tx
	otg       *ontology.Ontology
	otgWriter ontology.Writer
	table     *gorp.Table[Key, ICD]
	tasks     *task.Service
	uses      func(config msgpack.EncodedJSON) ([]Use, error)
}

// Create creates the ICD, or replaces it when an ICD with its key exists. A replacement
// must carry the Hash of the ICD as the writer read it, or Create returns an error
// wrapping library.ErrStale. Create assigns keys to the ICD, its entries, and their
// fields where absent, and sets Hash to the hash of the stored ICD. It returns a
// path-scoped validation error for each problem Check finds, and when a replacement
// drops a message or field that a task uses. Each task whose used entries change is
// rewritten, so its config carries their new hash.
func (w Writer) Create(ctx context.Context, ic *ICD) error {
	var prev *ICD
	if ic.Key != uuid.Nil() {
		var existing ICD
		err := w.table.NewRetrieve().
			Where(gorp.MatchKeys[Key, ICD](ic.Key)).
			Entry(&existing).
			Exec(ctx, w.tx)
		if err == nil {
			prev = &existing
		} else if !errors.Is(err, query.ErrNotFound) {
			return err
		}
	}
	if prev != nil {
		if err := library.CheckFresh(ic.Hash, prev.Hash); err != nil {
			return errors.Wrapf(err, "ICD %s", prev.Name)
		}
	} else if ic.Key == uuid.Nil() {
		ic.Key = uuid.New()
	}
	assignKeys(ic)
	if err := Check(*ic).Err(); err != nil {
		return err
	}
	var err error
	if ic.Hash, err = Hash(*ic); err != nil {
		return err
	}
	var stale []task.Task
	if prev != nil && prev.Hash != ic.Hash {
		if stale, err = w.staleTasks(ctx, *prev, *ic); err != nil {
			return err
		}
	}
	if err = w.table.NewCreate().Entry(ic).Exec(ctx, w.tx); err != nil {
		return err
	}
	if prev == nil {
		return w.otgWriter.DefineResources(ctx, ic.OntologyID())
	}
	tw := w.tasks.NewWriter(w.tx)
	for i := range stale {
		stale[i].Status = nil
		if err = tw.Create(ctx, &stale[i]); err != nil {
			return err
		}
	}
	return nil
}

// CreateMany creates or replaces each of the given ICDs as Create does.
func (w Writer) CreateMany(ctx context.Context, ics *[]ICD) error {
	for i := range *ics {
		if err := w.Create(ctx, &(*ics)[i]); err != nil {
			return err
		}
	}
	return nil
}

// Import replaces the entries of the ICD with the given key by those parsed from data
// in the given format. Entries and fields keep their keys when their names match (see
// Merge), and the tasks that use the ICD are rewritten as in Create. Import returns
// the updated ICD, a path-scoped validation error when data or the result is invalid,
// and an error wrapping query.ErrNotFound when no ICD has the key.
func (w Writer) Import(
	ctx context.Context,
	key Key,
	format Format,
	data []byte,
) (ICD, error) {
	entries, err := Parse(format, data)
	if err != nil {
		return ICD{}, err
	}
	var existing ICD
	if err = w.table.NewRetrieve().
		Where(gorp.MatchKeys[Key, ICD](key)).
		Entry(&existing).
		Exec(ctx, w.tx); err != nil {
		return ICD{}, err
	}
	ic := Merge(existing, entries)
	if err = w.Create(ctx, &ic); err != nil {
		return ICD{}, err
	}
	return ic, nil
}

// Rename changes the name of the ICD with the given key. It returns an error wrapping
// query.ErrNotFound when no ICD has the key.
func (w Writer) Rename(ctx context.Context, key Key, name string) error {
	if name == "" {
		return validate.PathedError(
			errors.Wrap(validate.ErrValidation, "name is required"),
			"name",
		)
	}
	var ic ICD
	if err := w.table.NewRetrieve().
		Where(gorp.MatchKeys[Key, ICD](key)).
		Entry(&ic).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	ic.Name = name
	return w.table.NewCreate().Entry(&ic).Exec(ctx, w.tx)
}

// Delete deletes the ICDs with the given keys. Delete is idempotent. It returns an
// error wrapping validate.ErrValidation when a task uses one of the ICDs.
func (w Writer) Delete(ctx context.Context, keys ...Key) error {
	for _, key := range keys {
		if err := library.CheckUnused(ctx, w.tx, w.otg, OntologyID(key)); err != nil {
			return err
		}
	}
	if err := w.table.NewDelete().
		Where(gorp.MatchKeys[Key, ICD](keys...)).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	return w.otgWriter.DeleteResources(ctx, OntologyIDs(keys)...)
}

// staleTasks returns the tasks that use next whose stamped hash no longer matches the
// entries they use. It returns a validation error when next drops a message or field
// that one of the tasks uses.
func (w Writer) staleTasks(ctx context.Context, prev, next ICD) ([]task.Task, error) {
	users, err := library.Users(ctx, w.tx, w.otg, next.OntologyID())
	if err != nil || len(users) == 0 {
		return nil, err
	}
	keys, err := task.KeysFromOntologyIDs(ontology.ResourceIDs(users))
	if err != nil {
		return nil, err
	}
	var tasks []task.Task
	if err = w.tasks.NewRetrieve().
		Where(task.MatchKeys(keys...)).
		Entries(&tasks).
		Exec(ctx, w.tx); err != nil {
		return nil, err
	}
	var stale []task.Task
	for _, t := range tasks {
		uses, err := w.uses(t.Config)
		if err != nil {
			return nil, err
		}
		if err = checkUses(t, uses, prev, next); err != nil {
			return nil, err
		}
		var ref Reference
		if err = t.Config.Unmarshal(&ref); err != nil {
			return nil, err
		}
		messages := lo.Map(uses, func(u Use, _ int) EntryKey { return u.Message })
		hash, err := TaskHash(next, messages)
		if err != nil {
			return nil, err
		}
		if hash != ref.ICDHash {
			stale = append(stale, t)
		}
	}
	return stale, nil
}

// checkUses returns a validation error, scoped to the "entries" path, when next lacks a
// message or field that the task uses. It names them from prev.
func checkUses(t task.Task, uses []Use, prev, next ICD) error {
	messages := messagesByKey(next)
	for _, u := range uses {
		m, ok := messages[u.Message]
		if !ok {
			return usesErrorf(
				"task %q uses message %q, which the ICD no longer has",
				t.Name,
				nameOf(prev, u.Message),
			)
		}
		fields := make(set.Set[FieldKey])
		for _, b := range m.Payload.FieldBases() {
			fields.Add(b.Key)
		}
		for _, f := range u.Fields {
			if !fields.Contains(f) {
				return usesErrorf(
					"task %q uses field %q of message %q, which the ICD no longer has",
					t.Name,
					fieldNameOf(prev, u.Message, f),
					m.Name,
				)
			}
		}
	}
	return nil
}

func usesErrorf(format string, args ...any) error {
	return validate.PathedError(
		errors.Wrapf(validate.ErrValidation, format, args...),
		"entries",
	)
}

func messagesByKey(ic ICD) map[EntryKey]MessageEntry {
	messages := make(map[EntryKey]MessageEntry)
	for _, e := range ic.Entries {
		if m, ok := e.Variant.(MessageEntry); ok {
			messages[m.Key] = m
		}
	}
	return messages
}

// nameOf returns the name of the message with the given key in ic, or the key when ic
// has no such message.
func nameOf(ic ICD, key EntryKey) string {
	if m, ok := messagesByKey(ic)[key]; ok {
		return m.Name
	}
	return key.String()
}

// fieldNameOf returns the name of the field with the given key in the message with the
// given key in ic, or the field key when ic has no such field.
func fieldNameOf(ic ICD, message EntryKey, field FieldKey) string {
	for _, b := range messagesByKey(ic)[message].Payload.FieldBases() {
		if b.Key == field {
			return b.Name
		}
	}
	return field.String()
}

// assignKeys gives a new key to every entry and field that has none. It skips entries
// and fields with no kind or encoding, which Check rejects.
func assignKeys(ic *ICD) {
	for i := range ic.Entries {
		e := &ic.Entries[i]
		if e.Variant == nil {
			continue
		}
		b := e.Base()
		if b.Key == uuid.Nil() {
			b.Key = uuid.New()
			e.SetBase(b)
		}
		m, ok := e.Variant.(MessageEntry)
		if !ok {
			continue
		}
		m.Payload.UpdateFieldBases(func(fb *BaseField) {
			if fb.Key == uuid.Nil() {
				fb.Key = uuid.New()
			}
		})
		e.Variant = m
	}
}
