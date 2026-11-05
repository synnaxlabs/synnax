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

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
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
	// status is used to write status updates.
	status status.Writer
	// table is the gorp table for rack entries.
	table *gorp.Table[Key, Rack]
	// driverRole is the role each new rack receives.
	driverRole role.Key
}

func resolveStatus(r *Rack) *Status {
	if r.Status == nil {
		return &Status{
			Key:     r.OntologyID().String(),
			Name:    r.Name,
			Time:    telem.Now(),
			Variant: status.VariantWarning,
			Message: "Rack status unknown",
			Details: StatusDetails{Rack: r.Key},
		}
	}
	stat := *r.Status
	stat.Key = r.OntologyID().String()
	stat.Details.Rack = r.Key
	stat.Name = r.Name
	return &stat
}

// writeStatus writes the status provided on r. With none, it keeps the stored row and
// changes only its name. The default status is written only if no row exists, because
// a stored row may be a live status from the Driver.
func (w Writer) writeStatus(ctx context.Context, r *Rack) error {
	stat := resolveStatus(r)
	if r.Status != nil {
		return w.status.Set(ctx, stat)
	}
	// A stored status can hold any details, so they do not decode as StatusDetails.
	var stored status.Status[any]
	err := gorp.NewRetrieve[status.Key, status.Status[any]]().
		Where(gorp.MatchKeys[status.Key, status.Status[any]](stat.Key)).
		Entry(&stored).
		Exec(ctx, w.tx)
	if errors.Is(err, query.ErrNotFound) {
		return w.status.Set(ctx, stat)
	}
	if err != nil || stored.Name == r.Name {
		return err
	}
	return w.status.Rename(ctx, stat.Key, r.Name)
}

// Create creates or updates a rack. If the rack key is zero or a rack with the key does
// not exist, a new rack will be created. A status provided on the rack replaces the
// stored status. With no provided status, the stored status stays and takes the rack's
// name, and only a rack with no stored status gets the default "unknown" status.
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
	exists, err := w.table.NewRetrieve().
		Where(gorp.MatchKeys[Key, Rack](r.Key)).
		Exists(ctx, w.tx)
	if err != nil {
		return err
	}
	if err = w.table.NewCreate().Entry(r).Exec(ctx, w.tx); err != nil {
		return err
	}
	otgID := r.OntologyID()
	if err = w.otg.DefineResources(ctx, otgID); err != nil {
		return err
	}
	if !exists {
		if err = w.otg.DefineRelationships(
			ctx,
			role.OntologyID(w.driverRole),
			ontology.RelationshipTypeParentOf,
			otgID,
		); err != nil {
			return err
		}
	}
	if err = w.writeStatus(ctx, r); err != nil {
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

// Rename renames the rack with the given key and its status, keeping every other field.
// It returns query.ErrNotFound if no rack with the key exists and a validation error if
// name is empty.
func (w Writer) Rename(ctx context.Context, key Key, name string) error {
	if err := w.table.NewUpdate().
		Where(gorp.MatchKeys[Key, Rack](key)).
		ChangeErr(func(_ gorp.Context, r Rack) (Rack, error) {
			r.Name = name
			if err := r.Validate(); err != nil {
				return Rack{}, err
			}
			return r, nil
		}).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	// A missing status row is valid: the next Create heals it.
	return errors.Skip(w.status.Rename(ctx, StatusKey(key), name), query.ErrNotFound)
}

// DeleteGuard deletes the rack with the given key and its associated status if the
// provided guard function returns nil.
func (w Writer) DeleteGuard(
	ctx context.Context,
	key Key,
	guard gorp.GuardFunc[Key, Rack],
) error {
	if err := w.table.NewDelete().
		Where(gorp.MatchKeys[Key, Rack](key)).
		Guard(guard).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	if err := w.otg.DeleteResources(ctx, key.OntologyID()); err != nil {
		return err
	}
	return w.status.Delete(ctx, key.OntologyID().String())
}
