// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package annotation

import (
	"context"

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/gorp"
)

// Writer is used to create, update, and delete annotations within Synnax. The writer
// executes all operations within the transaction provided to the Service.NewWriter
// method. If no transaction is provided, the writer will execute operations directly
// on the database.
type Writer struct {
	tx        gorp.Tx
	otgWriter ontology.Writer
	otg       *ontology.Ontology
}

// Create creates the given annotation. If the annotation does not have a key,
// a new key will be generated.
func (w Writer) Create(
	ctx context.Context,
	c *Annotation,
	parent ontology.ID,
) (err error) {
	var exists bool
	if c.Key == uuid.Nil {
		c.Key = uuid.New()
	} else {
		exists, err = gorp.NewRetrieve[uuid.UUID, Annotation]().WhereKeys(c.Key).Exists(ctx, w.tx)
		if err != nil {
			return
		}
	}
	if err = gorp.NewCreate[uuid.UUID, Annotation]().Entry(c).Exec(ctx, w.tx); err != nil {
		return
	}
	otgID := OntologyID(c.Key)
	if err = w.otgWriter.DefineResource(ctx, otgID); err != nil {
		return
	}
	if exists {
		if err = w.otgWriter.DeleteIncomingRelationshipsOfType(ctx, otgID, ontology.ParentOf); err != nil {
			return
		}
	}
	return w.otgWriter.DefineRelationship(ctx, parent, ontology.ParentOf, otgID)
}

// Delete deletes the annotations with the given keys.
func (w Writer) Delete(
	ctx context.Context,
	keys ...uuid.UUID,
) (err error) {
	if err = gorp.NewDelete[uuid.UUID, Annotation]().WhereKeys(keys...).Exec(ctx, w.tx); err != nil {
		return
	}
	for _, key := range keys {
		if err = w.otgWriter.DeleteResource(ctx, OntologyID(key)); err != nil {
			return
		}
	}
	return
}
