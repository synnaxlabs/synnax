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
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/gorp"
)

type Writer struct {
	tx    gorp.Tx
	otg   ontology.Writer
	group group.Group
	table *gorp.Table[Key, Project]
}

func (w Writer) Create(
	ctx context.Context,
	p *Project,
) (err error) {
	if p.Key == uuid.Nil {
		p.Key = uuid.New()
	}
	if err = p.Validate(); err != nil {
		return err
	}
	if err = w.table.NewCreate().Entry(p).Exec(ctx, w.tx); err != nil {
		return err
	}
	otgID := p.OntologyID()
	if err := w.otg.DefineResources(ctx, otgID); err != nil {
		return err
	}
	if err := w.otg.DefineRelationships(
		ctx,
		w.group.OntologyID(),
		ontology.RelationshipTypeParentOf,
		otgID,
	); err != nil {
		return err
	}
	return err
}

// CreateMany creates the given projects. If projects with the same key already
// exist, they will be overwritten.
func (w Writer) CreateMany(
	ctx context.Context,
	projects *[]Project,
) error {
	for i := range *projects {
		if err := w.Create(ctx, &(*projects)[i]); err != nil {
			return err
		}
	}
	return nil
}

func (w Writer) Rename(
	ctx context.Context,
	key Key,
	name string,
) error {
	return w.table.NewUpdate().Where(gorp.MatchKeys[Key, Project](key)).
		Change(func(_ gorp.Context, p Project) Project {
			p.Name = name
			return p
		}).Exec(ctx, w.tx)
}

func (w Writer) SetLayout(
	ctx context.Context,
	key Key,
	layout map[string]any,
) error {
	return w.table.NewUpdate().Where(gorp.MatchKeys[Key, Project](key)).
		Change(func(_ gorp.Context, p Project) Project {
			p.Layout = layout
			return p
		}).Exec(ctx, w.tx)
}

func (w Writer) Delete(
	ctx context.Context,
	keys ...Key,
) error {
	if err := w.table.NewDelete().
		Where(gorp.MatchKeys[Key, Project](keys...)).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	return w.otg.DeleteResources(ctx, OntologyIDs(keys)...)
}
