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
	"sync/atomic"

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/validate"
)

// Writer is used to create, update, and delete schematics within Synnax. The writer
// executes all operations within the transaction provided to the Service.NewWriter
// method. If no transaction is provided, the writer will execute operations directly
// on the database.
type Writer struct {
	tx        gorp.Tx
	otgWriter ontology.Writer
	otg       *ontology.Ontology
	table     *gorp.Table[Key, Schematic]
	// actionObserver is notified after a successful Dispatch so the cluster
	// signals subsystem can broadcast the action sequence on the schematic
	// channels. Nil when the service is opened without a Signals provider.
	actionObserver observe.Observer[ScopedAction]
	// seq points at the service-level monotonic sequence counter. Each
	// Dispatch increments it once and stamps the resulting value onto the
	// emitted ScopedAction so clients can dedupe echoes by ordering rather
	// than session identity.
	seq *atomic.Uint64
}

// Create creates the given schematic within the project provided. If the
// schematic does not have a key, a new key will be generated.
func (w Writer) Create(
	ctx context.Context,
	ws project.Key,
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
	if err = w.table.NewCreate().Entry(s).Exec(ctx, w.tx); err != nil {
		return
	}
	if exists {
		return
	}
	otgID := OntologyID(s.Key)
	if err := w.otgWriter.DefineResource(ctx, otgID); err != nil {
		return err
	}
	if ws == uuid.Nil {
		return nil
	}
	return w.otgWriter.DefineRelationship(
		ctx,
		project.OntologyID(ws),
		ontology.RelationshipTypeParentOf,
		otgID,
	)
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
	ws, ok, err := w.findParentProject(ctx, key)
	if err != nil || !ok {
		return err
	}
	if err := w.otgWriter.DefineResource(ctx, OntologyID(newKey)); err != nil {
		return err
	}
	// In the case of a snapshot, don't create a relationship to the project.
	if result.Snapshot {
		return nil
	}
	return w.otgWriter.DefineRelationship(
		ctx,
		project.OntologyID(ws),
		ontology.RelationshipTypeParentOf,
		OntologyID(newKey),
	)
}

// SetData replaces the body of the schematic with the given key with the
// provided value. Key, Name, and Snapshot are preserved from the existing
// entry; every other field on data overwrites the stored entry verbatim.
// Returns validate.ErrValidation when the target schematic is a snapshot,
// since snapshots are immutable.
func (w Writer) SetData(
	ctx context.Context,
	key Key,
	data Schematic,
) error {
	return w.table.NewUpdate().Where(gorp.MatchKeys[Key, Schematic](key)).
		ChangeErr(func(_ gorp.Context, s Schematic) (Schematic, error) {
			if s.Snapshot {
				return s, errors.Wrapf(validate.ErrValidation, "[Schematic] - cannot set data on snapshot %s:%s", key, s.Name)
			}
			data.Key = s.Key
			data.Name = s.Name
			data.Snapshot = s.Snapshot
			return data, nil
		}).Exec(ctx, w.tx)
}

// Dispatch applies a sequence of actions atomically to the schematic with the
// given key. After a successful update the actions are notified to the
// service-level observer so subscribers (cluster signals) can broadcast them.
// sessionKey identifies the originating client so subscribers can self-dedup.
// Snapshots are immutable except for Rename: returns validate.ErrValidation if
// the target is a snapshot and any action other than Rename is included.
func (w Writer) Dispatch(
	ctx context.Context,
	key Key,
	sessionKey string,
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
	if w.actionObserver != nil {
		w.actionObserver.Notify(ctx, ScopedAction{
			Key:        key,
			SessionKey: sessionKey,
			Seq:        w.seq.Add(1),
			Actions:    actions,
		})
	}
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
	for _, key := range keys {
		if err := w.otgWriter.DeleteResource(ctx, OntologyID(key)); err != nil {
			return err
		}
	}
	return nil
}
