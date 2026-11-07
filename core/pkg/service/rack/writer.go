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
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
)

// Writer is used to create, update, and delete racks within a Synnax cluster.
type Writer struct {
	svc *Service
	// tx is the underlying gorp transaction that rack operations will be executed
	// against.
	tx gorp.Tx
	// otg is a writer used to modify rack related resources and relationships within
	// the ontology.
	otg ontology.Writer
	// newKey returns a new key for a rack.
	newKey func(context.Context) (Key, error)
	// status is used to write status updates.
	status status.Writer
	// statuses retrieves the status that a rack holds.
	statuses *status.Service
	// table is the gorp table for rack entries.
	table *gorp.Table[Key, Rack]
	// driverRole is the role each new rack receives.
	driverRole role.Key
}

// inTx calls f with w, or with a Writer on a transaction of its own when w has no
// transaction.
func (w Writer) inTx(ctx context.Context, f func(Writer) error) error {
	db, ok := w.tx.(*gorp.DB)
	if !ok {
		return f(w)
	}
	return db.WithTx(ctx, func(tx gorp.Tx) error { return f(w.svc.NewWriter(tx)) })
}

func resolveStatus(r *Rack) *Status {
	if r.Status == nil {
		return &Status{
			Owner:   new(r.ResourceID()),
			Name:    r.Name,
			Time:    telem.Now(),
			Variant: status.VariantWarning,
			Message: "Rack status unknown",
		}
	}
	stat := *r.Status
	stat.Owner = new(r.ResourceID())
	stat.Name = r.Name
	return &stat
}

// writeStatus writes the status provided on r. With none, it keeps the stored status
// and changes only its name. The default status is written only if the rack holds no
// status, because a stored status may be a live status from the Driver.
func (w Writer) writeStatus(ctx context.Context, r *Rack) error {
	stat := resolveStatus(r)
	if r.Status != nil {
		return w.status.Set(ctx, stat)
	}
	// A stored status can hold any details, so they do not decode as StatusDetails.
	var held []status.Status[any]
	if err := w.statuses.NewRetrieve[any]().
		Where(status.MatchOwners[any](*stat.Owner)).
		Entries(&held).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	if len(held) == 0 {
		return w.status.Set(ctx, stat)
	}
	if held[0].Name == r.Name {
		return nil
	}
	return w.status.Rename(ctx, held[0].Key, r.Name)
}

// Create creates or updates a rack. If the rack key is zero or a rack with the key does
// not exist, a new rack will be created. An existing rack keeps its group. A status
// provided on the rack replaces the stored status. With no provided status, the stored
// status stays and takes the rack's name, and only a rack with no stored status gets
// the default "unknown" status.
func (w Writer) Create(ctx context.Context, r *Rack) error {
	return w.inTx(ctx, func(w Writer) error { return w.create(ctx, r) })
}

func (w Writer) create(ctx context.Context, r *Rack) error {
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
	var existing Rack
	err = w.table.NewRetrieve().
		Where(gorp.MatchKeys[Key, Rack](r.Key)).
		Entry(&existing).
		Exec(ctx, w.tx)
	if errors.Skip(err, query.ErrNotFound) != nil {
		return err
	}
	exists := err == nil
	if exists {
		r.Group = existing.Group
	} else if err = group.CheckMember(
		ctx,
		w.tx,
		r.Group,
		resource.TypeRack,
		nil,
	); err != nil {
		return err
	}
	if err = w.table.NewCreate().Entry(r).Exec(ctx, w.tx); err != nil {
		return err
	}
	otgID := r.ResourceID()
	if err = w.otg.DefineResources(ctx, otgID); err != nil {
		return err
	}
	if !exists {
		if err = w.otg.DefineRelationships(
			ctx,
			role.ResourceID(w.driverRole),
			ontology.RelationshipTypeParentOf,
			otgID,
		); err != nil {
			return err
		}
	}
	if err = w.writeStatus(ctx, r); err != nil {
		return err
	}
	return nil
}

// SetGroup moves the racks with keys into g, or out of their group when g is nil.
func (w Writer) SetGroup(ctx context.Context, g *group.Key, keys ...Key) error {
	return w.inTx(ctx, func(w Writer) error {
		if err := group.CheckMember(ctx, w.tx, g, resource.TypeRack, nil); err != nil {
			return err
		}
		return setGroup(w.table, g, keys...).Exec(ctx, w.tx)
	})
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
	return errors.Skip(
		w.status.RenameOwned(ctx, key.ResourceID(), name),
		query.ErrNotFound,
	)
}

// DeleteGuard deletes the rack with the given key, and with it its status, if the
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
	return w.otg.DeleteResources(ctx, key.ResourceID())
}
