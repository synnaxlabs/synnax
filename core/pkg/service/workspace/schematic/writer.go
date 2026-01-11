// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package schematic

import (
	"context"

	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/workspace"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/uuid"
	"github.com/synnaxlabs/x/validate"
)

// Writer is used to create, update, and delete logs within Synnax. The writer
// executes all operations within the transaction provided to the Service.NewWriter
// method. If no transaction is provided, the writer will execute operations directly
// on the database.
type Writer struct {
	tx        gorp.Tx
	otgWriter ontology.Writer
	otg       *ontology.Ontology
}

// Create creates the given log within the workspace provided. If the log does not
// have a key, a new key will be generated.
func (w Writer) Create(
	ctx context.Context,
	ws uuid.UUID,
	s *Schematic,
) (err error) {
	var exists bool
	if s.Key == uuid.Nil {
		s.Key = uuid.New()
	} else {
		exists, err = gorp.NewRetrieve[uuid.UUID, Schematic]().WhereKeys(s.Key).Exists(ctx, w.tx)
		if err != nil {
			return
		}
	}
	if err = gorp.NewCreate[uuid.UUID, Schematic]().Entry(s).Exec(ctx, w.tx); err != nil {
		return
	}
	if exists {
		return
	}
	otgID := OntologyID(s.Key)
	if err := w.otgWriter.DefineResource(ctx, otgID); err != nil {
		return err
	}
	return w.otgWriter.DefineRelationship(
		ctx,
		workspace.OntologyID(ws),
		ontology.ParentOf,
		otgID,
	)
}

func (w Writer) findParentWorkspace(ctx context.Context, key uuid.UUID) (uuid.UUID, bool, error) {
	var res []ontology.Resource
	if err := w.otg.NewRetrieve().
		WhereIDs(OntologyID(key)).
		TraverseTo(ontology.Parents).
		WhereTypes(workspace.OntologyType).
		Entries(&res).
		Exec(ctx, w.tx); err != nil {
		return uuid.Nil, false, err
	}
	if len(res) == 0 {
		return uuid.Nil, false, nil
	}
	k, err := uuid.Parse(res[0].ID.Key)
	return k, true, err
}

// Rename renames the log with the given key to the provided name.
func (w Writer) Rename(
	ctx context.Context,
	key uuid.UUID,
	name string,
) error {
	return gorp.NewUpdate[uuid.UUID, Schematic]().WhereKeys(key).
		Change(func(_ gorp.Context, s Schematic) Schematic {
			s.Name = name
			return s
		}).Exec(ctx, w.tx)
}

// Copy creates a copy of the log with the given key and name. If the snapshot flag is
// set to true, the copy will be a snapshot and will no longer be editable. The copied
// log will be bound into the result parameter.
func (w Writer) Copy(
	ctx context.Context,
	key uuid.UUID,
	name string,
	snapshot bool,
	result *Schematic,
) error {
	newKey := uuid.New()
	if err := gorp.NewUpdate[uuid.UUID, Schematic]().
		WhereKeys(key).
		Change(func(_ gorp.Context, s Schematic) Schematic {
			s.Key = newKey
			s.Name = name
			s.Snapshot = snapshot
			*result = s
			return s
		}).Exec(ctx, w.tx); err != nil {
		return err
	}
	ws, ok, err := w.findParentWorkspace(ctx, key)
	if err != nil || !ok {
		return err
	}
	if err := w.otgWriter.DefineResource(ctx, OntologyID(newKey)); err != nil {
		return err
	}
	// In the case of a snapshot, don't create a relationship to the workspace.
	if result.Snapshot {
		return nil
	}
	return w.otgWriter.DefineRelationship(
		ctx,
		workspace.OntologyID(ws),
		ontology.ParentOf,
		OntologyID(newKey),
	)
}

// SetData sets the data of the log with the given key to the provided data.
func (w Writer) SetData(
	ctx context.Context,
	key uuid.UUID,
	data map[string]any,
) error {
	return gorp.NewUpdate[uuid.UUID, Schematic]().WhereKeys(key).
		ChangeErr(func(_ gorp.Context, s Schematic) (Schematic, error) {
			if s.Snapshot {
				return s, errors.Wrapf(validate.Error, "[Schematic] - cannot set data on snapshot %s:%s", key, s.Name)
			}
			s.Data = data
			return s, nil
		}).Exec(ctx, w.tx)
}

// Delete deletes the logs with the given keys.
func (w Writer) Delete(
	ctx context.Context,
	keys ...uuid.UUID,
) error {
	err := gorp.NewDelete[uuid.UUID, Schematic]().WhereKeys(keys...).Exec(ctx, w.tx)
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
