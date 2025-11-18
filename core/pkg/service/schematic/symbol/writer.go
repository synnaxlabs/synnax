// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package symbol

import (
	"context"

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/gorp"
)

// Writer is used to create, update, and delete symbols within Synnax. The writer
// executes all operations within the transaction provided to the Service.NewWriter
// method. If no transaction is provided, the writer will execute operations directly
// on the database.
type Writer struct {
	tx        gorp.Tx
	otgWriter ontology.Writer
	otg       *ontology.Ontology
}

// Create creates the given symbol as a child of the ontology.Resource with the given
// parent ID. If the symbol does not have a key, a new key will be generated. If the symbol
// already exists, it will be updated and the existing parent relationship will be deleted
// and replaced with the new parent relationship.
func (w Writer) Create(
	ctx context.Context,
	s *Symbol,
	parent ontology.ID,
) (err error) {
	var exists bool
	if s.Key == uuid.Nil {
		s.Key = uuid.New()
	} else {
		exists, err = gorp.NewRetrieve[uuid.UUID, Symbol]().WhereKeys(s.Key).Exists(ctx, w.tx)
		if err != nil {
			return err
		}
	}
	if err = gorp.NewCreate[uuid.UUID, Symbol]().Entry(s).Exec(ctx, w.tx); err != nil {
		return err
	}
	otgID := OntologyID(s.Key)
	if err = w.otgWriter.DefineResource(ctx, otgID); err != nil {
		return err
	}
	// Symbol already exists = delete incoming relationships and define new parent
	// Symbol does not exist = define parent
	if exists {
		if err = w.otgWriter.DeleteIncomingRelationshipsOfType(ctx, otgID, ontology.ParentOf); err != nil {
			return err
		}
	}
	return w.otgWriter.DefineRelationship(ctx, parent, ontology.ParentOf, otgID)
}

// Rename renames the symbol with the given key to the provided name.
func (w Writer) Rename(
	ctx context.Context,
	key uuid.UUID,
	name string,
) error {
	return gorp.NewUpdate[uuid.UUID, Symbol]().WhereKeys(key).Change(func(_ gorp.Context, s Symbol) Symbol {
		s.Name = name
		return s
	}).Exec(ctx, w.tx)
}

// Delete deletes the symbols with the given keys.
func (w Writer) Delete(
	ctx context.Context,
	keys ...uuid.UUID,
) error {
	err := gorp.NewDelete[uuid.UUID, Symbol]().WhereKeys(keys...).Exec(ctx, w.tx)
	if err != nil {
		return err
	}
	for _, key := range keys {
		if err := w.otgWriter.DeleteResource(ctx, OntologyID(key)); err != nil {
			return err
		}
	}
	return nil
}
