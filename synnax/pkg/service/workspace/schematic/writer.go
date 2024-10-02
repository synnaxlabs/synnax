// Copyright 2023 Synnax Labs, Inc.
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

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/workspace"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/validate"
)

type Writer struct {
	tx        gorp.Tx
	otgWriter ontology.Writer
	otg       *ontology.Ontology
}

func (w Writer) Create(
	ctx context.Context,
	ws uuid.UUID,
	s *Schematic,
) (err error) {
	if s.Key == uuid.Nil {
		s.Key = uuid.New()
	}
	if err = gorp.NewCreate[uuid.UUID, Schematic]().Entry(s).Exec(ctx, w.tx); err != nil {
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

func (w Writer) Rename(
	ctx context.Context,
	key uuid.UUID,
	name string,
) error {
	return gorp.NewUpdate[uuid.UUID, Schematic]().WhereKeys(key).Change(func(p Schematic) Schematic {
		p.Name = name
		return p
	}).Exec(ctx, w.tx)
}

func (w Writer) Copy(
	ctx context.Context,
	key uuid.UUID,
	name string,
	snapshot bool,
	result *Schematic,
) error {
	newKey := uuid.New()
	if err := gorp.NewUpdate[uuid.UUID, Schematic]().WhereKeys(key).Change(func(s Schematic) Schematic {
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

func (w Writer) SetData(
	ctx context.Context,
	key uuid.UUID,
	data string,
) error {
	return gorp.NewUpdate[uuid.UUID, Schematic]().WhereKeys(key).ChangeErr(func(s Schematic) (Schematic, error) {
		if s.Snapshot {
			return s, errors.Wrapf(validate.Error, "[Schematic] - cannot set data on snapshot %s:%s", key, s.Name)
		}
		s.Data = data
		return s, nil
	}).Exec(ctx, w.tx)
}

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
