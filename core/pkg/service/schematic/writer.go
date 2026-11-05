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
	"uuid"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
)

// Writer is used to create, update, and delete schematics within Synnax. The writer
// executes all operations within the transaction provided to the Service.NewWriter
// method. If no transaction is provided, the writer will execute operations directly
// on the database.
type Writer struct {
	tx         gorp.Tx
	otgWriter  ontology.Writer
	otg        *ontology.Ontology
	table      *gorp.Table[Key, Schematic]
	dispatcher actions.Dispatcher[Key, Action]
}

// WithDispatchKey returns a copy of the writer that stamps dispatchKey onto the create
// broadcasts it emits, so the client that sent the create recognizes its own echo.
func (w Writer) WithDispatchKey(dispatchKey string) Writer {
	w.dispatcher = w.dispatcher.WithDispatchKey(dispatchKey)
	return w
}

// Create creates the given schematic within the project provided. If the
// schematic does not have a key, a new key will be generated. An existing schematic
// keeps its group.
func (w Writer) Create(
	ctx context.Context,
	projectKey project.Key,
	s *Schematic,
) (err error) {
	var exists bool
	if s.Key == uuid.Nil() {
		s.Key = uuid.New()
	} else {
		var stored Schematic
		err = w.table.NewRetrieve().
			Where(gorp.MatchKeys[Key, Schematic](s.Key)).
			Entry(&stored).
			Exec(ctx, w.tx)
		if errors.Skip(err, query.ErrNotFound) != nil {
			return err
		}
		if exists = err == nil; exists {
			s.Group = stored.Group
		}
	}
	if !exists {
		if err = group.CheckMember(
			ctx,
			w.tx,
			s.Group,
			resource.TypeSchematic,
			nil,
		); err != nil {
			return err
		}
	}
	s.ApplyDefaults()
	if err = s.Validate(); err != nil {
		return err
	}
	if err = w.table.NewCreate().Entry(s).Exec(ctx, w.tx); err != nil {
		return err
	}
	if !exists {
		otgID := s.ResourceID()
		if err := w.otgWriter.DefineResources(ctx, otgID); err != nil {
			return err
		}
		if projectKey != uuid.Nil() {
			if err := w.otgWriter.DefineRelationships(
				ctx,
				project.ResourceID(projectKey),
				ontology.RelationshipTypeParentOf,
				otgID,
			); err != nil {
				return err
			}
		}
	}
	w.dispatcher.NotifyOnCommit(
		ctx,
		w.tx,
		s.Key,
		NewCreateAction(CreatePayload{Schematic: *s}),
	)
	return nil
}

// SetGroup moves the schematics with keys into g, or out of their group when g is nil.
func (w Writer) SetGroup(ctx context.Context, g *group.Key, keys ...Key) error {
	if err := group.CheckMember(ctx, w.tx, g, resource.TypeSchematic, nil); err != nil {
		return err
	}
	if err := setGroup(w.table, g, keys...).Exec(ctx, w.tx); err != nil {
		return err
	}
	for _, k := range keys {
		w.dispatcher.NotifyOnCommit(
			ctx,
			w.tx,
			k,
			NewSetGroupAction(SetGroupPayload{Group: g}),
		)
	}
	return nil
}

// CreateMany creates the given schematics within the project provided. If schematics
// with the same key already exist, they will be overwritten.
func (w Writer) CreateMany(
	ctx context.Context,
	projectKey project.Key,
	schematics *[]Schematic,
) error {
	for i := range *schematics {
		if err := w.Create(ctx, projectKey, &(*schematics)[i]); err != nil {
			return err
		}
	}
	return nil
}

func (w Writer) findParentProject(
	ctx context.Context,
	key Key,
) (project.Key, bool, error) {
	var res []ontology.Resource
	if err := w.otg.NewRetrieve().
		WhereIDs(ResourceID(key)).
		TraverseTo(ontology.ParentsTraverser).
		WhereTypes(resource.TypeProject).
		Entries(&res).
		Exec(ctx, w.tx); err != nil {
		return uuid.Nil(), false, err
	}
	if len(res) == 0 {
		return uuid.Nil(), false, nil
	}
	k, err := uuid.Parse(res[0].ID.Key)
	return k, true, err
}

// Copy creates a copy of the schematic with the given key and name. If the
// snapshot flag is set to true, the copy will be a snapshot and will no
// longer be editable. The copy sits in no group. The copied schematic will be bound
// into the result parameter.
func (w Writer) Copy(
	ctx context.Context,
	key Key,
	name string,
	snapshot bool,
	result *Schematic,
) error {
	newKey := uuid.New()
	if err := w.table.NewUpdate().
		Where(gorp.MatchKeys[Key, Schematic](key)).
		Change(func(_ gorp.Context, s Schematic) Schematic {
			s.Key = newKey
			s.Name = name
			s.Snapshot = snapshot
			s.Group = nil
			*result = s
			return s
		}).Exec(ctx, w.tx); err != nil {
		return err
	}
	projectKey, ok, err := w.findParentProject(ctx, key)
	if err != nil || !ok {
		return err
	}
	if err := w.otgWriter.DefineResources(ctx, ResourceID(newKey)); err != nil {
		return err
	}
	// In the case of a snapshot, don't create a relationship to the project.
	if result.Snapshot {
		return nil
	}
	return w.otgWriter.DefineRelationships(
		ctx,
		project.ResourceID(projectKey),
		ontology.RelationshipTypeParentOf,
		ResourceID(newKey),
	)
}

// Delete deletes the schematics with the given keys.
func (w Writer) Delete(
	ctx context.Context,
	keys ...Key,
) error {
	err := w.table.NewDelete().
		Where(gorp.MatchKeys[Key, Schematic](keys...)).
		Exec(ctx, w.tx)
	if err != nil {
		return err
	}
	return w.otgWriter.DeleteResources(ctx, ResourceIDs(keys)...)
}
