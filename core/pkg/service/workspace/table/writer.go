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

	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/workspace"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/uuid"
)

// Writer is used to create, update, and delete tables within Synnax. The writer
// executes all operations within the transaction provided to the Service.NewWriter
// method. If no transaction is provided, the writer will execute operations directly
// on the database.
type Writer struct {
	tx        gorp.Tx
	otgWriter ontology.Writer
	otg       *ontology.Ontology
}

// Create creates the given table within the workspace provided. If the table does not
// have a key, a new key will be generated.
func (w Writer) Create(
	ctx context.Context,
	ws uuid.UUID,
	s *Table,
) (err error) {
	var exists bool
	if s.Key == uuid.Nil {
		s.Key = uuid.New()
	} else {
		exists, err = gorp.NewRetrieve[uuid.UUID, Table]().WhereKeys(s.Key).Exists(ctx, w.tx)
		if err != nil {
			return
		}
	}
	if err = gorp.NewCreate[uuid.UUID, Table]().Entry(s).Exec(ctx, w.tx); err != nil {
		return
	}
	if exists {
		return
	}
	otgID := OntologyID(s.Key)
	if err = w.otgWriter.DefineResource(ctx, otgID); err != nil {
		return
	}
	return w.otgWriter.DefineRelationship(
		ctx,
		workspace.OntologyID(ws),
		ontology.ParentOf,
		otgID,
	)
}

// Rename renames the table with the given key to the provided name.
func (w Writer) Rename(
	ctx context.Context,
	key uuid.UUID,
	name string,
) error {
	return gorp.NewUpdate[uuid.UUID, Table]().
		WhereKeys(key).
		Change(func(_ gorp.Context, t Table) Table {
			t.Name = name
			return t
		}).
		Exec(ctx, w.tx)
}

// SetData sets the data of the table with the given key to the provided data.
func (w Writer) SetData(
	ctx context.Context,
	key uuid.UUID,
	data map[string]any,
) error {
	return gorp.NewUpdate[uuid.UUID, Table]().
		WhereKeys(key).
		Change(func(_ gorp.Context, t Table) Table {
			t.Data = data
			return t
		}).Exec(ctx, w.tx)
}

// Delete deletes the tables with the given keys.
func (w Writer) Delete(
	ctx context.Context,
	keys ...uuid.UUID,
) (err error) {
	if err = gorp.NewDelete[uuid.UUID, Table]().WhereKeys(keys...).Exec(ctx, w.tx); err != nil {
		return
	}
	for _, key := range keys {
		if err = w.otgWriter.DeleteResource(ctx, OntologyID(key)); err != nil {
			return
		}
	}
	return
}
