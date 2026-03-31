// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package project

import (
	"context"

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/distribution/group"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/gorp"
)

type Writer struct {
	tx    gorp.Tx
	otg   ontology.Writer
	group group.Group
	table *gorp.Table[uuid.UUID, Project]
}

func (w Writer) Create(
	ctx context.Context,
	p *Project,
) (err error) {
	if p.Key == uuid.Nil {
		p.Key = uuid.New()
	}
	if err = w.table.NewCreate().Entry(p).Exec(ctx, w.tx); err != nil {
		return
	}
	otgID := OntologyID(p.Key)
	if err := w.otg.DefineResource(ctx, otgID); err != nil {
		return err
	}
	if err := w.otg.DefineRelationship(
		ctx,
		w.group.OntologyID(),
		ontology.RelationshipTypeParentOf,
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
	return w.table.NewUpdate().
		WhereKeys(key).
		Change(func(_ gorp.Context, p Project) Project {
			p.Name = name
			return p
		}).Exec(ctx, w.tx)
}

func (w Writer) Delete(
	ctx context.Context,
	keys ...uuid.UUID,
) error {
	if err := w.table.NewDelete().WhereKeys(keys...).Exec(ctx, w.tx); err != nil {
		return err
	}
	for _, key := range keys {
		if err := w.otg.DeleteResource(ctx, OntologyID(key)); err != nil {
			return err
		}
	}
	return nil
}
