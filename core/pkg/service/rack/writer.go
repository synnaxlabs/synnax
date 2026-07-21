// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package rack

import (
	"context"

	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/telem"
)

// Writer is used to create, update, and delete racks within a Synnax cluster.
type Writer struct {
	// tx is the underlying gorp transaction that rack operations will be executed
	// against.
	tx gorp.Tx
	// otg is a writer used to modify rack related resources and relationships within
	// the ontology.
	otg ontology.Writer
	// group is the base group that racks will be created under.
	group group.Group
	// newKey returns a new key for a rack.
	newKey func(context.Context) (Key, error)
	// newTaskKey returns a new key for a task within the rack.
	newTaskKey func(context.Context, Key) (uint32, error)
	// status is used to write status updates.
	status status.Writer[StatusDetails]
	// table is the gorp table for rack entries.
	table *gorp.Table[Key, Rack]
}

func resolveStatus(r *Rack) *Status {
	if r.Status == nil {
		return &Status{
			Key:     r.Key.OntologyID().String(),
			Name:    r.Name,
			Time:    telem.Now(),
			Variant: status.VariantWarning,
			Message: "Status unknown",
			Details: StatusDetails{Rack: r.Key},
		}
	}
	stat := *r.Status
	stat.Key = r.Key.OntologyID().String()
	stat.Details.Rack = r.Key
	stat.Name = r.Name
	return &stat
}

// healStatus restores a rack's status row if it has gone missing (e.g. deleted
// out-of-band) without clobbering a live one. Racks are re-created on every scan cycle,
// so on a no-op update the default "unknown" status must not overwrite a status the
// driver has already reported; it is only written when no row exists.
func (w Writer) healStatus(
	ctx context.Context,
	stat *Status,
) error {
	if exists, err := gorp.NewRetrieve[string, Status]().
		Where(gorp.MatchKeys[string, Status](stat.Key)).
		Exists(ctx, w.tx); err != nil || exists {
		return err
	}
	return w.status.Set(ctx, stat)
}

// Create creates or updates a rack. If the rack key is zero or a rack with the key
// does not exist, a new rack will be created. If a status is provided on the rack,
// it will be used instead of the default "unknown" status.
func (w Writer) Create(ctx context.Context, r *Rack) error {
	var err error
	if r.Key.IsZero() {
		r.Key, err = w.newKey(ctx)
		if err != nil {
			return err
		}
	}
	if err = r.Validate(); err != nil {
		return err
	}
	if err = w.table.NewCreate().Entry(r).Exec(ctx, w.tx); err != nil {
		return err
	}
	otgID := r.Key.OntologyID()
	if err = w.otg.DefineResources(ctx, otgID); err != nil {
		return err
	}
	stat := resolveStatus(r)
	if r.Status != nil {
		if err = w.status.Set(ctx, stat); err != nil {
			return err
		}
	} else if err = w.healStatus(ctx, stat); err != nil {
		return err
	}
	return w.otg.DefineRelationships(
		ctx, w.group.OntologyID(), ontology.RelationshipTypeParentOf, otgID,
	)
}

// CreateMany creates the given racks. If racks with the same key already exist, they
// will be overwritten.
func (w Writer) CreateMany(ctx context.Context, racks *[]Rack) error {
	for i := range *racks {
		if err := w.Create(ctx, &(*racks)[i]); err != nil {
			return err
		}
	}
	return nil
}

// Delete deletes the rack with the provided key and its associated status. Delete is
// idempotent, and deleting a non-existent rack will not return an error.
func (w Writer) Delete(ctx context.Context, key Key) error {
	return w.DeleteGuard(ctx, key, nil)
}

// DeleteGuard deletes the rack with the given key and its associated status if the
// provided guard function returns nil.
func (w Writer) DeleteGuard(ctx context.Context, key Key, guard gorp.GuardFunc[Key, Rack]) error {
	if err := w.table.NewDelete().Where(gorp.MatchKeys[Key, Rack](key)).Guard(guard).Exec(ctx, w.tx); err != nil {
		return err
	}
	return w.status.Delete(ctx, key.OntologyID().String())
}

// NewTaskKey returns a new, unique key for the task on the provided rack.
func (w Writer) NewTaskKey(ctx context.Context, key Key) (next uint32, err error) {
	return w.newTaskKey(ctx, key)
}
