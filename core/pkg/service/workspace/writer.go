// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package workspace

import (
	"context"

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/x/gorp"
)

type Writer struct {
	tx    gorp.Tx
	otg   ontology.Writer
	group group.Group
}

func (w Writer) Create(
	ctx context.Context,
	ws *Workspace,
) (err error) {
	if ws.Key == uuid.Nil {
		ws.Key = uuid.New()
	}
	if err = gorp.NewCreate[uuid.UUID, Workspace]().Entry(ws).Exec(ctx, w.tx); err != nil {
		return
	}
	otgID := OntologyID(ws.Key)
	if err := w.otg.DefineResource(ctx, otgID); err != nil {
		return err
	}
	if err := w.otg.DefineRelationship(
		ctx,
		w.group.OntologyID(),
		ontology.ParentOf,
		otgID,
	); err != nil {
		return err
	}
	if err := w.otg.DefineRelationship(
		ctx,
		user.OntologyID(ws.Author),
		ontology.ParentOf,
		otgID,
	); err != nil {
		return err
	}
	return err
}

func (w Writer) Rename(
	ctx context.Context,
	key uuid.UUID,
	name string,
) error {
	return gorp.NewUpdate[uuid.UUID, Workspace]().
		WhereKeys(key).
		Change(func(_ gorp.Context, ws Workspace) Workspace {
			ws.Name = name
			return ws
		}).Exec(ctx, w.tx)
}

func (w Writer) SetLayout(
	ctx context.Context,
	key uuid.UUID,
	layout string,
) error {
	return gorp.NewUpdate[uuid.UUID, Workspace]().
		WhereKeys(key).
		Change(func(_ gorp.Context, ws Workspace) Workspace {
			ws.Layout = layout
			return ws
		}).Exec(ctx, w.tx)
}

func (w Writer) Delete(
	ctx context.Context,
	keys ...uuid.UUID,
) error {
	if err := gorp.NewDelete[uuid.UUID, Workspace]().WhereKeys(keys...).Exec(ctx, w.tx); err != nil {
		return err
	}
	for _, key := range keys {
		if err := w.otg.DeleteResource(ctx, OntologyID(key)); err != nil {
			return err
		}
	}
	return nil
}
