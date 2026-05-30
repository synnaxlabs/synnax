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

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/workspace"
	"github.com/synnaxlabs/x/gorp"
)

// Writer is used to create, update, and delete tables within Synnax. The writer
// executes all operations within the transaction provided to the Service.NewWriter
// method. If no transaction is provided, the writer will execute operations directly
// on the database.
type Writer struct {
	tx         gorp.Tx
	otgWriter  ontology.Writer
	otg        *ontology.Ontology
	tbl        *gorp.Table[Key, Table]
	dispatcher actions.Dispatcher[Key, Action]
}

// Create creates the given table within the workspace provided. If the table does not
// have a key, a new key will be generated.
func (w Writer) Create(
	ctx context.Context,
	ws workspace.Key,
	s *Table,
) (err error) {
	var exists bool
	if s.Key == uuid.Nil {
		s.Key = uuid.New()
	} else {
		exists, err = w.tbl.NewRetrieve().Where(gorp.MatchKeys[Key, Table](s.Key)).Exists(ctx, w.tx)
		if err != nil {
			return
		}
	}
	if err = w.tbl.NewCreate().Entry(s).Exec(ctx, w.tx); err != nil {
		return
	}
	if exists {
		return
	}
	otgID := OntologyID(s.Key)
	if err = w.otgWriter.DefineResource(ctx, otgID); err != nil {
		return
	}
	if ws == uuid.Nil {
		return nil
	}
	return w.otgWriter.DefineRelationship(
		ctx,
		workspace.OntologyID(ws),
		ontology.RelationshipTypeParentOf,
		otgID,
	)
}

// Rename renames the table with the given key to the provided name.
func (w Writer) Rename(
	ctx context.Context,
	key Key,
	name string,
) error {
	return w.tbl.NewUpdate().
		Where(gorp.MatchKeys[Key, Table](key)).
		Change(func(_ gorp.Context, t Table) Table {
			t.Name = name
			return t
		}).
		Exec(ctx, w.tx)
}

// SetData replaces the body of the table with the given key with the provided
// value. Key and Name are preserved from the existing entry; Rows, Columns,
// and Cells on data overwrite the stored entry verbatim.
func (w Writer) SetData(
	ctx context.Context,
	key Key,
	data Table,
) error {
	return w.tbl.NewUpdate().
		Where(gorp.MatchKeys[Key, Table](key)).
		Change(func(_ gorp.Context, t Table) Table {
			data.Key = t.Key
			data.Name = t.Name
			return data
		}).Exec(ctx, w.tx)
}

// Dispatch applies a sequence of actions atomically to the table with the
// given key. After a successful update the actions are notified to the
// service-level observer so subscribers (cluster signals) can broadcast them.
// dispatchKey is a client-generated identifier carried verbatim onto the
// broadcast so the originating client can match its own echo against the set
// of outstanding local replays and skip a redundant reduce when no foreign
// action interleaved.
func (w Writer) Dispatch(
	ctx context.Context,
	key Key,
	dispatchKey string,
	actions []Action,
) error {
	if err := w.tbl.NewUpdate().Where(gorp.MatchKeys[Key, Table](key)).
		ChangeErr(func(_ gorp.Context, t Table) (Table, error) {
			return Reduce(t, actions...)
		}).Exec(ctx, w.tx); err != nil {
		return err
	}
	w.dispatcher.Notify(ctx, key, dispatchKey, actions)
	return nil
}

// Delete deletes the tables with the given keys.
func (w Writer) Delete(
	ctx context.Context,
	keys ...Key,
) (err error) {
	if err = w.tbl.NewDelete().Where(gorp.MatchKeys[Key, Table](keys...)).Exec(ctx, w.tx); err != nil {
		return
	}
	for _, key := range keys {
		if err = w.otgWriter.DeleteResource(ctx, OntologyID(key)); err != nil {
			return
		}
	}
	return
}
