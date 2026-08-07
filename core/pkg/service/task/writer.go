// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package task

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task/common"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

type Writer struct {
	tx        gorp.Tx
	otgWriter ontology.Writer
	otg       *ontology.Ontology
	group     group.Group
	status    status.Writer[StatusDetails]
	table     *gorp.Table[Key, Task]
	configs   common.ConfigRegistry
}

// resolveStatus returns the status to persist for t. Without a provided status, a
// task created for the first time provably has no live instance, so it gets a quiet
// "has not been deployed" placeholder; an existing task may be running, so its
// placeholder stays a "status unknown" warning.
func resolveStatus(t *Task, provided *Status, existed bool) *Status {
	if provided == nil {
		message := fmt.Sprintf("%s status unknown", t.Name)
		variant := status.VariantWarning
		if !existed {
			message = fmt.Sprintf("%s has not been deployed", t.Name)
			variant = status.VariantDisabled
		}
		return &Status{
			Key:     t.OntologyID().String(),
			Time:    telem.Now(),
			Name:    t.Name,
			Message: message,
			Variant: variant,
			Details: StatusDetails{Task: t.Key},
		}
	}
	provided.Key = t.OntologyID().String()
	provided.Details.Task = t.Key
	provided.Name = t.Name
	return provided
}

// healStatus returns the task's effective status, writing stat first if no row exists.
// Tasks are re-created on every scan cycle, so on a no-op update the placeholder must
// not overwrite a status the driver has already reported.
func (w Writer) healStatus(ctx context.Context, stat *Status) (*Status, error) {
	var existing Status
	err := gorp.NewRetrieve[string, Status]().
		Where(gorp.MatchKeys[string, Status](stat.Key)).
		Entry(&existing).
		Exec(ctx, w.tx)
	if err == nil {
		return &existing, nil
	}
	if !errors.Is(err, query.ErrNotFound) {
		return nil, err
	}
	if err := w.status.Set(ctx, stat); err != nil {
		return nil, err
	}
	return stat, nil
}

// configRecord returns the key of the config record parented to the task with the
// given ontology ID whose resource type matches taskType, and false when the task
// has no such record.
func (w Writer) configRecord(
	id ontology.ID,
	taskType string,
) (uuid.UUID, bool, error) {
	parents, err := w.otg.RetrieveParents(w.tx, id)
	if err != nil {
		return uuid.Nil, false, err
	}
	recordID, ok := lo.Find(parents[id], func(p ontology.ID) bool {
		return p.Type == ontology.ResourceType(taskType)
	})
	if !ok {
		return uuid.Nil, false, nil
	}
	key, err := uuid.Parse(recordID.Key)
	if err != nil {
		return uuid.Nil, false, err
	}
	return key, true, nil
}

// Create creates or updates a task. The task's config is stored as a record in its
// type's config store; the task row itself carries only the config hash. A provided
// status is persisted as given. Without one, a new task gets a "has not been
// deployed" placeholder, and an existing task keeps its reported status (healed to
// "status unknown" if the row is missing).
func (w Writer) Create(ctx context.Context, t *Task) error {
	if t.Key == uuid.Nil {
		t.Key = uuid.New()
	}
	store, ok := w.configs.Store(ontology.ResourceType(t.Type))
	if !ok {
		return errors.Wrapf(validate.ErrValidation, "unknown task type %q", t.Type)
	}
	var existing Task
	existed := true
	if err := w.table.NewRetrieve().
		Where(gorp.MatchKeys[Key, Task](t.Key)).
		Entry(&existing).
		Exec(ctx, w.tx); err != nil {
		if !errors.Is(err, query.ErrNotFound) {
			return err
		}
		existed = false
	}
	otgID := t.OntologyID()
	recordKey, recordExists := uuid.Nil, false
	if existed {
		var err error
		if recordKey, recordExists, err = w.configRecord(
			otgID, existing.Type,
		); err != nil {
			return err
		}
	}
	if existed && existing.Snapshot {
		t.ConfigHash = existing.ConfigHash
		if recordExists {
			existingStore, ok := w.configs.Store(
				ontology.ResourceType(existing.Type),
			)
			if !ok {
				return errors.Wrapf(
					validate.ErrValidation, "unknown task type %q", existing.Type,
				)
			}
			data, err := existingStore.Read(ctx, w.tx, recordKey)
			if err != nil {
				return err
			}
			t.Config = data
		}
	} else {
		typeChanged := recordExists && existing.Type != t.Type
		oldRecordKey := recordKey
		if !recordExists || typeChanged {
			recordKey = uuid.New()
		}
		if err := store.Write(ctx, w.tx, recordKey, t.Config); err != nil {
			return err
		}
		// Read the record back so the returned config and its hash reflect the
		// canonical stored shape, defaults included.
		canonical, err := store.Read(ctx, w.tx, recordKey)
		if err != nil {
			return err
		}
		t.Config = canonical
		if t.ConfigHash, err = hashConfig(configContent(canonical)); err != nil {
			return err
		}
		// Delete the old type's record only after the new config is stored, so a
		// rejected config cannot destroy the prior one.
		if typeChanged {
			oldStore, ok := w.configs.Store(ontology.ResourceType(existing.Type))
			if ok {
				if err := oldStore.Delete(ctx, w.tx, oldRecordKey); err != nil {
					return err
				}
			}
		}
		recordExists = true
	}
	config := t.Config // Restored after the row write; rows do not store config.
	providedStatus := t.Status
	t.Config, t.Status = nil, nil
	err := w.table.NewCreate().Entry(t).Exec(ctx, w.tx)
	t.Config = config
	if err != nil {
		return err
	}
	stat := resolveStatus(t, providedStatus, existed)
	if providedStatus != nil {
		if err := w.status.Set(ctx, stat); err != nil {
			return err
		}
	} else if stat, err = w.healStatus(ctx, stat); err != nil {
		return err
	}
	t.Status = stat
	exists, err := w.otg.NewRetrieve().WhereIDs(otgID).Exists(ctx, w.tx)
	if err != nil {
		return err
	}
	if !exists {
		if err = w.otgWriter.DefineResources(ctx, otgID); err != nil {
			return err
		}
	}
	if recordExists {
		if err = w.otgWriter.DefineRelationships(
			ctx,
			ontology.ID{
				Type: ontology.ResourceType(t.Type),
				Key:  recordKey.String(),
			},
			ontology.RelationshipTypeParentOf,
			otgID,
		); err != nil {
			return err
		}
	}
	// Internal tasks get no group parent, keeping them out of the resource tree.
	if t.Internal || exists {
		return nil
	}
	return w.otgWriter.DefineRelationships(
		ctx,
		w.group.OntologyID(),
		ontology.RelationshipTypeParentOf,
		otgID,
	)
}

// CreateMany creates the given tasks. If tasks with the same key already exist, they
// will be overwritten.
func (w Writer) CreateMany(ctx context.Context, tasks *[]Task) error {
	for i := range *tasks {
		if err := w.Create(ctx, &(*tasks)[i]); err != nil {
			return err
		}
	}
	return nil
}

// Delete deletes the task with the given key, its config record, and its associated
// status.
func (w Writer) Delete(ctx context.Context, key Key, allowInternal bool) error {
	var t Task
	found := true
	if err := w.table.NewRetrieve().
		Where(gorp.MatchKeys[Key, Task](key)).
		Entry(&t).
		Exec(ctx, w.tx); err != nil {
		if !errors.Is(err, query.ErrNotFound) {
			return err
		}
		found = false
	}
	if found {
		recordKey, recordExists, err := w.configRecord(OntologyID(key), t.Type)
		if err != nil {
			return err
		}
		if recordExists {
			if store, ok := w.configs.Store(
				ontology.ResourceType(t.Type),
			); ok {
				if err := store.Delete(ctx, w.tx, recordKey); err != nil {
					return err
				}
			}
		}
	}
	if err := w.table.NewDelete().
		Where(gorp.MatchKeys[Key, Task](key)).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	if err := w.otgWriter.DeleteResources(ctx, OntologyID(key)); err != nil {
		return err
	}
	return w.status.Delete(ctx, OntologyID(key).String())
}

func (w Writer) Copy(
	ctx context.Context,
	key Key,
	name string,
	snapshot bool,
) (Task, error) {
	newKey := uuid.New()
	var res Task
	if err := w.table.NewUpdate().
		Where(gorp.MatchKeys[Key, Task](key)).
		Change(func(_ gorp.Context, t Task) Task {
			t.Key = newKey
			t.Name = name
			t.Snapshot = snapshot
			res = t
			return t
		}).
		Exec(ctx, w.tx); err != nil {
		return Task{}, err
	}
	if err := w.status.Set(ctx, resolveStatus(&res, nil, false)); err != nil {
		return Task{}, err
	}
	if err := w.otgWriter.DefineResources(ctx, OntologyID(newKey)); err != nil {
		return Task{}, err
	}
	recordKey, recordExists, err := w.configRecord(OntologyID(key), res.Type)
	if err != nil {
		return Task{}, err
	}
	store, ok := w.configs.Store(ontology.ResourceType(res.Type))
	if !recordExists || !ok {
		return res, nil
	}
	newRecordKey := uuid.New()
	if err := store.Copy(ctx, w.tx, recordKey, newRecordKey); err != nil {
		return Task{}, err
	}
	if err := w.otgWriter.DefineRelationships(
		ctx,
		ontology.ID{
			Type: ontology.ResourceType(res.Type),
			Key:  newRecordKey.String(),
		},
		ontology.RelationshipTypeParentOf,
		OntologyID(newKey),
	); err != nil {
		return Task{}, err
	}
	data, err := store.Read(ctx, w.tx, newRecordKey)
	if err != nil {
		return Task{}, err
	}
	res.Config = data
	return res, nil
}
