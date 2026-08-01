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

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/validate"
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

// Create creates the given schematic within the project provided. If the
// schematic does not have a key, a new key will be generated.
func (w Writer) Create(
	ctx context.Context,
	projectKey project.Key,
	s *Schematic,
) (err error) {
	var exists bool
	if s.Key == uuid.Nil {
		s.Key = uuid.New()
	} else {
		exists, err = w.table.NewRetrieve().Where(gorp.MatchKeys[Key, Schematic](s.Key)).Exists(ctx, w.tx)
		if err != nil {
			return
		}
	}
	if err = s.Validate(); err != nil {
		return
	}
	if err = w.table.NewCreate().Entry(s).Exec(ctx, w.tx); err != nil {
		return
	}
	if !exists {
		otgID := s.OntologyID()
		if err := w.otgWriter.DefineResources(ctx, otgID); err != nil {
			return err
		}
		if projectKey != uuid.Nil {
			if err := w.otgWriter.DefineRelationships(
				ctx,
				project.OntologyID(projectKey),
				ontology.RelationshipTypeParentOf,
				otgID,
			); err != nil {
				return err
			}
		}
	}
	// Notify last: a create rejected by ontology validation must not be broadcast.
	w.dispatcher.Notify(
		ctx, s.Key, "", []Action{NewCreateAction(CreatePayload{Schematic: *s})},
	)
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

func (w Writer) findParentProject(ctx context.Context, key Key) (project.Key, bool, error) {
	var res []ontology.Resource
	if err := w.otg.NewRetrieve().
		WhereIDs(OntologyID(key)).
		TraverseTo(ontology.ParentsTraverser).
		WhereTypes(ontology.ResourceTypeProject).
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

// Copy creates a copy of the schematic with the given key and name. If the
// snapshot flag is set to true, the copy will be a snapshot and will no
// longer be editable. The copied schematic will be bound into the result
// parameter.
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
			*result = s
			return s
		}).Exec(ctx, w.tx); err != nil {
		return err
	}
	projectKey, ok, err := w.findParentProject(ctx, key)
	if err != nil || !ok {
		return err
	}
	if err := w.otgWriter.DefineResources(ctx, OntologyID(newKey)); err != nil {
		return err
	}
	// In the case of a snapshot, don't create a relationship to the project.
	if result.Snapshot {
		return nil
	}
	return w.otgWriter.DefineRelationships(
		ctx,
		project.OntologyID(projectKey),
		ontology.RelationshipTypeParentOf,
		OntologyID(newKey),
	)
}

// Dispatch applies a sequence of actions atomically to the schematic with the
// given key. After a successful update the actions are notified to the
// service-level observer so subscribers (cluster signals) can broadcast them.
// dispatchKey is a client-generated identifier carried verbatim onto the
// broadcast so the originating client can match its own echo against the set
// of outstanding local replays and skip a redundant reduce when no foreign
// action interleaved. Snapshots are immutable except for Rename: returns
// validate.ErrValidation if the target is a snapshot and any action other
// than Rename is included.
func (w Writer) Dispatch(
	ctx context.Context,
	key Key,
	dispatchKey string,
	actions []Action,
) error {
	if err := w.table.NewUpdate().Where(gorp.MatchKeys[Key, Schematic](key)).
		ChangeErr(func(_ gorp.Context, s Schematic) (Schematic, error) {
			if s.Snapshot {
				for _, a := range actions {
					if a.Type != ActionTypeRename {
						return s, errors.Wrapf(
							validate.ErrValidation,
							"[Schematic] - cannot dispatch %s on snapshot %s:%s",
							a.Type,
							key,
							s.Name,
						)
					}
				}
			}
			return Reduce(s, actions...)
		}).Exec(ctx, w.tx); err != nil {
		return err
	}
	w.dispatcher.Notify(ctx, key, dispatchKey, actions)
	return nil
}

// Delete deletes the schematics with the given keys.
func (w Writer) Delete(
	ctx context.Context,
	keys ...Key,
) error {
	err := w.table.NewDelete().Where(gorp.MatchKeys[Key, Schematic](keys...)).Exec(ctx, w.tx)
	if err != nil {
		return err
	}
	return w.otgWriter.DeleteResources(ctx, OntologyIDs(keys)...)
}
