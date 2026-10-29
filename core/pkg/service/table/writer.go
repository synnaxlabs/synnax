// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package table

import (
	"context"
	"uuid"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/validate"
)

// Writer is used to create, update, and delete tables within Synnax. The writer
// executes all operations within the transaction provided to the Service.NewWriter
// method. If no transaction is provided, the writer will execute operations directly on
// the database.
type Writer struct {
	tx         gorp.Tx
	otgWriter  ontology.Writer
	otg        *ontology.Ontology
	tbl        *gorp.Table[Key, Table]
	dispatcher actions.Dispatcher[Key, Action]
}

// WithDispatchKey returns a copy of the writer that stamps dispatchKey onto the create
// broadcasts it emits, so the client that sent the create recognizes its own echo.
func (w Writer) WithDispatchKey(dispatchKey string) Writer {
	w.dispatcher = w.dispatcher.WithDispatchKey(dispatchKey)
	return w
}

// Create creates the given table within the project provided. If the table does not
// have a key, a new key will be generated.
func (w Writer) Create(ctx context.Context, projectKey project.Key, t *Table) error {
	var (
		exists bool
		err    error
	)
	if t.Key == uuid.Nil() {
		t.Key = uuid.New()
	} else {
		exists, err = w.tbl.NewRetrieve().
			Where(gorp.MatchKeys[Key, Table](t.Key)).
			Exists(ctx, w.tx)
		if err != nil {
			return err
		}
	}
	if err = t.Validate(); err != nil {
		return err
	}
	if err = validateUniqueKeys(*t); err != nil {
		return err
	}
	if err = w.tbl.NewCreate().Entry(t).Exec(ctx, w.tx); err != nil {
		return err
	}
	if !exists {
		otgID := t.OntologyID()
		if err = w.otgWriter.DefineResources(ctx, otgID); err != nil {
			return err
		}
		if projectKey != uuid.Nil() {
			if err = w.otgWriter.DefineRelationships(
				ctx,
				project.OntologyID(projectKey),
				ontology.RelationshipTypeParentOf,
				otgID,
			); err != nil {
				return err
			}
		}
	}
	w.dispatcher.NotifyOnCommit(
		ctx,
		w.tx,
		t.Key,
		NewCreateAction(CreatePayload{Table: *t}),
	)
	return nil
}

// CreateMany creates the given tables within the project provided. If tables with the
// same key already exist, they will be overwritten.
func (w Writer) CreateMany(
	ctx context.Context,
	projectKey project.Key,
	tables *[]Table,
) error {
	for i := range *tables {
		if err := w.Create(ctx, projectKey, &(*tables)[i]); err != nil {
			return err
		}
	}
	return nil
}

// Delete deletes the tables with the given keys.
func (w Writer) Delete(ctx context.Context, keys ...Key) error {
	if err := w.tbl.NewDelete().
		Where(gorp.MatchKeys[Key, Table](keys...)).Exec(ctx, w.tx); err != nil {
		return err
	}
	return w.otgWriter.DeleteResources(ctx, OntologyIDs(keys)...)
}

func validateUniqueKeys(t Table) error {
	if k, ok := duplicateKey(t.Rows, func(r Row) string { return r.Key }); ok {
		return errors.Wrapf(validate.ErrValidation, "duplicate row key %q", k)
	}
	if k, ok := duplicateKey(t.Columns, func(c Column) string { return c.Key }); ok {
		return errors.Wrapf(validate.ErrValidation, "duplicate column key %q", k)
	}
	return nil
}

func duplicateKey[T any](items []T, keyOf func(T) string) (string, bool) {
	seen := make(set.Set[string], len(items))
	for _, item := range items {
		k := keyOf(item)
		if seen.Contains(k) {
			return k, true
		}
		seen.Add(k)
	}
	return "", false
}
