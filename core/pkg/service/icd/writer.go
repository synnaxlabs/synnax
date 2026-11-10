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
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/library"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/validate"
)

// reader reads ICDs and the tasks that use them through a snapshot.
type reader struct {
	snap  gorp.Snapshot
	table *Table
	tasks *task.Service
	uses  func(config msgpack.EncodedJSON) ([]Use, error)
}

// Writer creates, updates, and deletes ICDs within a transaction.
type Writer struct {
	reader
	tx gorp.Tx
}

// Create creates the ICD, or replaces it when an ICD with its key exists. A replacement
// must carry the Hash of the ICD as the writer read it, or Create returns an error
// wrapping library.ErrStale. Create assigns keys to the ICD, its messages, and their
// fields where absent, and sets Hash to the hash of the stored ICD. It returns a
// path-scoped validation error for each problem Check finds, and when a replacement
// drops a message or field that a task uses. Each task that is not a snapshot and whose
// used messages change gets their new hash in its config.
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
	var stale []staleTask
	if prev != nil && prev.Hash != ic.Hash {
		if stale, err = w.staleTasks(ctx, *prev, *ic); err != nil {
			return err
		}
	}
	if err = w.table.table.NewCreate().Entry(ic).Exec(ctx, w.tx); err != nil {
		return err
	}
	tw := w.tasks.NewWriter(w.tx)
	for _, t := range stale {
		if _, err = tw.Apply(ctx, t.key, task.NewSetAction(task.SetPayload{
			Path:  []string{"config", "icd_hash"},
			Value: t.hash,
		})); err != nil {
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

// check returns every problem Create would reject ic for, without storing it: each
// problem the package-level Check finds, and each message or field that a task uses and
// ic lacks.
func (r reader) check(ctx context.Context, ic ICD) (library.Problems, error) {
	problems := Check(ic)
	if ic.Key == uuid.Nil() {
		return problems, nil
	}
	prev, err := r.retrieve(ctx, ic.Key)
	if errors.Is(err, query.ErrNotFound) {
		return problems, nil
	}
	if err != nil {
		return nil, err
	}
	users, err := r.users(ctx, prev)
	if err != nil {
		return nil, err
	}
	for _, u := range users {
		problems = append(problems, u.missing(prev, ic)...)
	}
	return problems, nil
}

// Import replaces the messages of the ICD with the given key by those parsed from data
// in the given format. Messages and fields keep their keys when their names match (see
// Merge), and the tasks that use the ICD are rewritten as in Create. Import returns
// the updated ICD, a path-scoped validation error when data or the result is invalid,
// and an error wrapping query.ErrNotFound when no ICD has the key.
func (w Writer) Import(
	ctx context.Context,
	key Key,
	format Format,
	data []byte,
) (ICD, error) {
	messages, err := Parse(format, data)
	if err != nil {
		return ICD{}, err
	}
	existing, err := w.retrieve(ctx, key)
	if err != nil {
		return ICD{}, err
	}
	ic := Merge(existing, messages)
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
	return w.table.table.NewCreate().Entry(&ic).Exec(ctx, w.tx)
}

// Delete deletes the ICDs with the given keys. Delete is idempotent. It returns a
// gorp.RestrictError when a task uses one of the ICDs.
func (w Writer) Delete(ctx context.Context, keys ...Key) error {
	return w.table.table.NewDelete().
		Where(gorp.MatchKeys[Key, ICD](keys...)).
		Exec(ctx, w.tx)
}

func (r reader) retrieve(ctx context.Context, key Key) (ICD, error) {
	var ic ICD
	err := r.table.table.NewRetrieve().
		Where(gorp.MatchKeys[Key, ICD](key)).
		Entry(&ic).
		Exec(ctx, r.snap)
	return ic, err
}

// user is a task that uses an ICD, with the messages it uses.
type user struct {
	task task.Task
	uses []Use
}

// users returns each task that uses ic.
func (r reader) users(ctx context.Context, ic ICD) ([]user, error) {
	keys, err := r.table.taskKeys(ctx, r.snap, ic.Key)
	if err != nil || len(keys) == 0 {
		return nil, err
	}
	var tasks []task.Task
	if err = r.tasks.NewRetrieve().
		Where(task.MatchKeys(keys...)).
		Entries(&tasks).
		Exec(ctx, r.snap); err != nil {
		return nil, err
	}
	users := make([]user, len(tasks))
	for i, t := range tasks {
		uses, err := r.uses(t.Config)
		if err != nil {
			return nil, err
		}
		users[i] = user{task: t, uses: uses}
	}
	return users, nil
}

// missing returns a problem for each message or field that u uses and next lacks,
// named from prev. A missing message is a problem with the messages, and a missing
// field one with the fields of its message.
func (u user) missing(prev, next ICD) library.Problems {
	indexes := make(map[MessageKey]int, len(next.Messages))
	for i, m := range next.Messages {
		indexes[m.Key] = i
	}
	var problems library.Problems
	for _, use := range u.uses {
		i, ok := indexes[use.Message]
		if !ok {
			problems.Addf(
				[]string{"messages"},
				"task %q uses message %q, which the ICD no longer has",
				u.task.Name,
				nameOf(prev, use.Message),
			)
			continue
		}
		m := next.Messages[i]
		fields := make(set.Set[FieldKey])
		for _, b := range m.Payload.FieldBases() {
			fields.Add(b.Key)
		}
		for _, f := range use.Fields {
			if !fields.Contains(f) {
				problems.Addf(
					[]string{"messages", strconv.Itoa(i), "payload", "fields"},
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

// staleTask is a task whose stamped hash no longer matches the messages it uses.
type staleTask struct {
	key  task.Key
	hash string
}

// staleTasks returns the tasks that use next, are not snapshots, and whose stamped hash
// no longer matches the messages they use, each with the hash it needs. It returns a
// path-scoped validation error for each message or field a task uses and next lacks.
func (w Writer) staleTasks(ctx context.Context, prev, next ICD) ([]staleTask, error) {
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
	var stale []staleTask
	for _, u := range users {
		if u.task.Snapshot {
			continue
		}
		var ref Reference
		if err = u.task.Config.Unmarshal(&ref); err != nil {
			return nil, err
		}
		messages := lo.Map(
			u.uses,
			func(use Use, _ int) MessageKey { return use.Message },
		)
		hash, err := TaskHash(next, messages)
		if err != nil {
			return nil, err
		}
		if hash != ref.ICDHash {
			stale = append(stale, staleTask{key: u.task.Key, hash: hash})
		}
	}
	return stale, nil
}

// messageOf returns the message with the given key in ic.
func messageOf(ic ICD, key MessageKey) (Message, bool) {
	return lo.Find(ic.Messages, func(m Message) bool { return m.Key == key })
}

// nameOf returns the name of the message with the given key in ic, or the key when ic
// has no such message.
func nameOf(ic ICD, key MessageKey) string {
	if m, ok := messageOf(ic, key); ok {
		return m.Name
	}
	return key.String()
}

// fieldNameOf returns the name of the field with the given key in the message with the
// given key in ic, or the field key when ic has no such field.
func fieldNameOf(ic ICD, message MessageKey, field FieldKey) string {
	m, _ := messageOf(ic, message)
	for _, b := range m.Payload.FieldBases() {
		if b.Key == field {
			return b.Name
		}
	}
	return field.String()
}

// assignKeys gives a new key to every message and field that has none. It skips text
// fields with no encoding, which Check rejects.
func assignKeys(ic *ICD) {
	for i := range ic.Messages {
		m := &ic.Messages[i]
		if m.Key == uuid.Nil() {
			m.Key = uuid.New()
		}
		m.Payload.UpdateFieldBases(func(fb *BaseField) {
			if fb.Key == uuid.Nil() {
				fb.Key = uuid.New()
			}
		})
	}
}
