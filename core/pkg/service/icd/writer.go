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
	"strconv"
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
		existing, err := w.retrieve(ctx, ic.Key)
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

// Check returns every problem Create would reject ic for, without storing it: each
// problem the package-level Check finds, and each message or field that a task uses and
// ic lacks.
func (w Writer) Check(ctx context.Context, ic ICD) (library.Problems, error) {
	problems := Check(ic)
	if ic.Key == uuid.Nil() {
		return problems, nil
	}
	prev, err := w.retrieve(ctx, ic.Key)
	if errors.Is(err, query.ErrNotFound) {
		return problems, nil
	}
	if err != nil {
		return nil, err
	}
	users, err := w.users(ctx, prev)
	if err != nil {
		return nil, err
	}
	for _, u := range users {
		problems = append(problems, u.missing(prev, ic)...)
	}
	return problems, nil
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
	existing, err := w.retrieve(ctx, key)
	if err != nil {
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
	ic, err := w.retrieve(ctx, key)
	if err != nil {
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

func (w Writer) retrieve(ctx context.Context, key Key) (ICD, error) {
	var ic ICD
	err := w.table.NewRetrieve().
		Where(gorp.MatchKeys[Key, ICD](key)).
		Entry(&ic).
		Exec(ctx, w.tx)
	return ic, err
}

// user is a task that uses an ICD, with the entries it uses.
type user struct {
	task task.Task
	uses []Use
}

// users returns each task that uses ic.
func (w Writer) users(ctx context.Context, ic ICD) ([]user, error) {
	resources, err := library.Users(ctx, w.tx, w.otg, ic.OntologyID())
	if err != nil || len(resources) == 0 {
		return nil, err
	}
	keys, err := task.KeysFromOntologyIDs(ontology.ResourceIDs(resources))
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
	users := make([]user, len(tasks))
	for i, t := range tasks {
		uses, err := w.uses(t.Config)
		if err != nil {
			return nil, err
		}
		users[i] = user{task: t, uses: uses}
	}
	return users, nil
}

// missing returns a problem for each message or field that u uses and next lacks,
// named from prev. A missing message is a problem with the entries, and a missing field
// one with the fields of its message.
func (u user) missing(prev, next ICD) library.Problems {
	indexes := make(map[EntryKey]int)
	for i, e := range next.Entries {
		if m, ok := e.Variant.(MessageEntry); ok {
			indexes[m.Key] = i
		}
	}
	var problems library.Problems
	for _, use := range u.uses {
		i, ok := indexes[use.Message]
		if !ok {
			problems.Addf(
				[]string{"entries"},
				"task %q uses message %q, which the ICD no longer has",
				u.task.Name,
				nameOf(prev, use.Message),
			)
			continue
		}
		m := next.Entries[i].Variant.(MessageEntry)
		fields := make(set.Set[FieldKey])
		for _, b := range m.Payload.FieldBases() {
			fields.Add(b.Key)
		}
		for _, f := range use.Fields {
			if !fields.Contains(f) {
				problems.Addf(
					[]string{"entries", strconv.Itoa(i), "payload", "fields"},
					"task %q uses field %q of message %q, which the ICD no longer has",
					u.task.Name,
					fieldNameOf(prev, use.Message, f),
					m.Name,
				)
			}
		}
	}
	return problems
}

// staleTasks returns the tasks that use next whose stamped hash no longer matches the
// entries they use. It returns a path-scoped validation error for each message or field
// a task uses and next lacks.
func (w Writer) staleTasks(ctx context.Context, prev, next ICD) ([]task.Task, error) {
	users, err := w.users(ctx, next)
	if err != nil {
		return nil, err
	}
	var missing library.Problems
	for _, u := range users {
		missing = append(missing, u.missing(prev, next)...)
	}
	if err = missing.Err(); err != nil {
		return nil, err
	}
	var stale []task.Task
	for _, u := range users {
		var ref Reference
		if err = u.task.Config.Unmarshal(&ref); err != nil {
			return nil, err
		}
		messages := lo.Map(u.uses, func(use Use, _ int) EntryKey { return use.Message })
		hash, err := TaskHash(next, messages)
		if err != nil {
			return nil, err
		}
		if hash != ref.ICDHash {
			stale = append(stale, u.task)
		}
	}
	return stale, nil
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
