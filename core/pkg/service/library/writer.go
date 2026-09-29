// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package library

import (
	"context"
	"strings"
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/service/library/icd"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/validate"
)

// Writer creates, updates, and deletes libraries within a transaction.
type Writer struct {
	tx        gorp.Tx
	otg       *ontology.Ontology
	otgWriter ontology.Writer
	table     *gorp.Table[Key, Library]
	tasks     *task.Service
}

// Create creates the library, or replaces it when a library with its key exists. Keys
// are assigned to the library, its entries, and their fields where absent. When a
// replacement changes the library's hash, every task that uses it is rewritten so its
// config carries the new hash. Create returns a path-scoped validation error when the
// library is invalid.
func (w Writer) Create(ctx context.Context, l *Library) error {
	var prev *Library
	if l.Key == uuid.Nil() {
		l.Key = uuid.New()
	} else {
		var existing Library
		err := w.table.NewRetrieve().
			Where(gorp.MatchKeys[Key, Library](l.Key)).
			Entry(&existing).
			Exec(ctx, w.tx)
		if err == nil {
			prev = &existing
		} else if !errors.Is(err, query.ErrNotFound) {
			return err
		}
	}
	assignKeys(l)
	l.ApplyDefaults()
	if err := l.Validate(); err != nil {
		return err
	}
	if err := validateEntries(l.Entries); err != nil {
		return err
	}
	if err := w.table.NewCreate().Entry(l).Exec(ctx, w.tx); err != nil {
		return err
	}
	if prev == nil {
		return w.otgWriter.DefineResources(ctx, l.OntologyID())
	}
	prevHash, err := Hash(*prev)
	if err != nil {
		return err
	}
	hash, err := Hash(*l)
	if err != nil || hash == prevHash {
		return err
	}
	return w.restampUsers(ctx, l.Key)
}

// CreateMany creates or replaces each of the given libraries.
func (w Writer) CreateMany(ctx context.Context, libs *[]Library) error {
	for i := range *libs {
		if err := w.Create(ctx, &(*libs)[i]); err != nil {
			return err
		}
	}
	return nil
}

// ImportICD replaces the entries of the library with the given key by those parsed
// from data, an interface control document in the given format. Entries and fields
// keep their keys when their names match (see icd.Merge), and every task that uses the
// library is rewritten as in Create. It returns the updated library, a path-scoped
// validation error when data or the result is invalid, and query.ErrNotFound when no
// library has the key.
func (w Writer) ImportICD(
	ctx context.Context,
	key Key,
	format icd.Format,
	data []byte,
) (Library, error) {
	entries, err := icd.Parse(format, data)
	if err != nil {
		return Library{}, err
	}
	var existing Library
	if err = w.table.NewRetrieve().
		Where(gorp.MatchKeys[Key, Library](key)).
		Entry(&existing).
		Exec(ctx, w.tx); err != nil {
		return Library{}, err
	}
	l := icd.Merge(existing, entries)
	if err = w.Create(ctx, &l); err != nil {
		return Library{}, err
	}
	return l, nil
}

// Rename changes the name of the library with the given key.
func (w Writer) Rename(ctx context.Context, key Key, name string) error {
	if name == "" {
		return validate.PathedError(
			errors.Wrap(validate.ErrValidation, "name is required"),
			"name",
		)
	}
	return w.table.NewUpdate().
		Where(gorp.MatchKeys[Key, Library](key)).
		Change(func(_ gorp.Context, l Library) Library {
			l.Name = name
			return l
		}).
		Exec(ctx, w.tx)
}

// Delete deletes the libraries with the given keys. Delete is idempotent. It returns
// an error wrapping validate.ErrValidation when a task uses one of the libraries.
func (w Writer) Delete(ctx context.Context, keys ...Key) error {
	for _, key := range keys {
		users, err := w.retrieveUsers(ctx, key)
		if err != nil {
			return err
		}
		if len(users) > 0 {
			names := lo.Map(users, func(r ontology.Resource, _ int) string {
				return r.Name
			})
			return errors.Wrapf(
				validate.ErrValidation,
				"library %s is used by %s",
				key,
				strings.Join(names, ", "),
			)
		}
	}
	if err := w.table.NewDelete().
		Where(gorp.MatchKeys[Key, Library](keys...)).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	return w.otgWriter.DeleteResources(ctx, OntologyIDs(keys)...)
}

// restampUsers rewrites each task that uses the library through the task writer, whose
// config store stamps the library's new hash into the task config.
func (w Writer) restampUsers(ctx context.Context, key Key) error {
	users, err := w.retrieveUsers(ctx, key)
	if err != nil || len(users) == 0 {
		return err
	}
	taskKeys, err := task.KeysFromOntologyIDs(ontology.ResourceIDs(users))
	if err != nil {
		return err
	}
	var tasks []task.Task
	if err = w.tasks.NewRetrieve().
		Where(task.MatchKeys(taskKeys...)).
		Entries(&tasks).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	tw := w.tasks.NewWriter(w.tx)
	for i := range tasks {
		tasks[i].Status = nil
		if err = tw.Create(ctx, &tasks[i]); err != nil {
			return err
		}
	}
	return nil
}

func (w Writer) retrieveUsers(
	ctx context.Context,
	key Key,
) ([]ontology.Resource, error) {
	var users []ontology.Resource
	if err := w.otg.NewRetrieve().
		WhereIDs(OntologyID(key)).
		TraverseTo(ontology.UsersTraverser).
		WhereTypes(ontology.ResourceTypeTask).
		Entries(&users).
		Exec(ctx, w.tx); err != nil && !errors.Is(err, query.ErrNotFound) {
		return nil, err
	}
	return users, nil
}

// assignKeys gives a new key to every entry and field that has none. It skips entries
// and fields with no kind or encoding, which validation rejects.
func assignKeys(l *Library) {
	for i := range l.Entries {
		e := &l.Entries[i]
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
		for j := range m.Fields {
			f := &m.Fields[j]
			if f.Variant == nil {
				continue
			}
			if fb := f.Base(); fb.Key == uuid.Nil() {
				fb.Key = uuid.New()
				f.SetBase(fb)
			}
		}
	}
}
