// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package log

import (
	"context"

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/workspace"
	"github.com/synnaxlabs/x/gorp"
)

type Writer struct {
	tx        gorp.Tx
	otgWriter ontology.Writer
	otg       *ontology.Ontology
}

func (w Writer) Create(
	ctx context.Context,
	ws uuid.UUID,
	s *Log,
) (err error) {
	if s.Key == uuid.Nil {
		s.Key = uuid.New()
	}
	if err = gorp.NewCreate[uuid.UUID, Log]().Entry(s).Exec(ctx, w.tx); err != nil {
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
	return gorp.NewUpdate[uuid.UUID, Log]().WhereKeys(key).Change(func(p Log) Log {
		p.Name = name
		return p
	}).Exec(ctx, w.tx)
}

func (w Writer) SetData(
	ctx context.Context,
	key uuid.UUID,
	data string,
) error {
	return gorp.NewUpdate[uuid.UUID, Log]().WhereKeys(key).ChangeErr(func(s Log) (Log, error) {
		s.Data = data
		return s, nil
	}).Exec(ctx, w.tx)
}

func (w Writer) Delete(
	ctx context.Context,
	keys ...uuid.UUID,
) error {
	err := gorp.NewDelete[uuid.UUID, Log]().WhereKeys(keys...).Exec(ctx, w.tx)
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
