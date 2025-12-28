// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc

import (
	"context"

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/gorp"
)

// Writer is used to create, update, and delete arcs within Synnax. The writer
// executes all operations within the transaction provided to the Service.NewWriter
// method. If no transaction is provided, the writer will execute operations directly
// on the database.
type Writer struct {
	tx  gorp.Tx
	otg ontology.Writer
}

// Create creates the given Arc. If the Arc does not have a key,
// a new key will be generated.
func (w Writer) Create(
	ctx context.Context,
	a *Arc,
) error {
	var (
		exists bool
		err    error
	)
	if a.Key == uuid.Nil {
		a.Key = uuid.New()
	} else {
		exists, err = gorp.NewRetrieve[uuid.UUID, Arc]().WhereKeys(a.Key).Exists(ctx, w.tx)
		if err != nil {
			return err
		}
	}
	if err = gorp.NewCreate[uuid.UUID, Arc]().Entry(a).Exec(ctx, w.tx); err != nil {
		return err
	}
	otgID := OntologyID(a.Key)
	if !exists {
		if err = w.otg.DefineResource(ctx, otgID); err != nil {
			return err
		}
	}
	return nil
}

// Delete deletes the arcs with the given keys.
func (w Writer) Delete(
	ctx context.Context,
	keys ...uuid.UUID,
) (err error) {
	if err = gorp.NewDelete[uuid.UUID, Arc]().WhereKeys(keys...).Exec(ctx, w.tx); err != nil {
		return
	}
	for _, key := range keys {
		if err = w.otg.DeleteResource(ctx, OntologyID(key)); err != nil {
			return
		}
	}
	return
}
