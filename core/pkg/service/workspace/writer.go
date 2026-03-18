// Copyright 2026 Synnax Labs, Inc.
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

// ChildDeleter deletes child resources of a specific type from a workspace.
// Each deleter is associated with an ontology type, and only receives keys
// that match that type during cascade deletion. Services like schematic.Service,
// lineplot.Service, etc. should implement this interface.
type ChildDeleter interface {
	// Type returns the ontology type this deleter handles.
	Type() ontology.Type
	// DeleteChildren removes the resources with the given keys within the
	// provided transaction.
	DeleteChildren(ctx context.Context, tx gorp.Tx, keys ...uuid.UUID) error
}

type Writer struct {
	tx            gorp.Tx
	otg           ontology.Writer
	otgR          *ontology.Ontology
	group         group.Group
	childDeleters []ChildDeleter
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
		ontology.RelationshipTypeParentOf,
		otgID,
	); err != nil {
		return err
	}
	if err := w.otg.DefineRelationship(
		ctx,
		user.OntologyID(ws.Author),
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
	for _, key := range keys {
		if err := w.deleteChildren(ctx, key); err != nil {
			return err
		}
	}
	if err := gorp.NewDelete[uuid.UUID, Workspace]().
		WhereKeys(keys...).Exec(ctx, w.tx); err != nil {
		return err
	}
	for _, key := range keys {
		if err := w.otg.DeleteResource(ctx, OntologyID(key)); err != nil {
			return err
		}
	}
	return nil
}

func (w Writer) deleteChildren(ctx context.Context, key uuid.UUID) error {
	var children []ontology.Resource
	if err := w.otgR.NewRetrieve().
		WhereIDs(OntologyID(key)).
		TraverseTo(ontology.ChildrenTraverser).
		Entries(&children).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	if len(children) == 0 {
		return nil
	}
	byType := make(map[ontology.Type][]uuid.UUID)
	for _, child := range children {
		k, err := uuid.Parse(child.ID.Key)
		if err != nil {
			return err
		}
		byType[child.ID.Type] = append(byType[child.ID.Type], k)
	}
	// Children with no registered deleter are silently skipped. Ensure a
	// ChildDeleter is registered for every type that can be a workspace child.
	for _, deleter := range w.childDeleters {
		keys, ok := byType[deleter.Type()]
		if !ok || len(keys) == 0 {
			continue
		}
		if err := deleter.DeleteChildren(ctx, w.tx, keys...); err != nil {
			return err
		}
	}
	return nil
}
