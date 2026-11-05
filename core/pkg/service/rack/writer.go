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
	"github.com/synnaxlabs/x/validate"
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

// healStatus restores a rack's status row if it has gone missing (e.g. deleted
// out-of-band) without clobbering a live one. Racks are re-created on every scan cycle,
// so on a no-op update the default "unknown" status must not overwrite a status the
// driver has already reported; it is only written when no row exists.
func (w Writer) healStatus(
	ctx context.Context,
	stat *Status,
) error {
	var held []Status
	if err := w.statuses.NewRetrieve[struct{}]().
		Where(status.MatchOwners[struct{}](*stat.Owner)).
		Entries(&held).
		Exec(ctx, w.tx); err != nil || len(held) > 0 {
		return err
	}
	return w.status.Set(ctx, stat)
}

// Create creates or updates a rack. If the rack key is zero or a rack with the key
// does not exist, a new rack will be created. An existing rack keeps its group. If a
// status is provided on the rack, it will be used instead of the default "unknown"
// status.
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
	stat := resolveStatus(r)
	if r.Status != nil {
		if err = w.status.Set(ctx, stat); err != nil {
			return err
		}
	} else if err = w.healStatus(ctx, stat); err != nil {
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

// Delete deletes the racks with keys, their statuses, and every task they hold. It
// returns a validation error when a rack is embedded, holds a device, or fails guard.
// Delete racks through task.Writer.DeleteRacks, which keeps the tasks of users.
func (w Writer) Delete(
	ctx context.Context,
	guard gorp.GuardFunc[Key, Rack],
	keys ...Key,
) error {
	if err := w.table.NewDelete().
		Where(gorp.MatchKeys[Key, Rack](keys...)).
		Guard(guardEmbedded).
		Guard(guard).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	return w.otg.DeleteResources(ctx, ResourceIDs(keys)...)
}

func guardEmbedded(_ gorp.Context, r Rack) error {
	if !r.Embedded {
		return nil
	}
	return errors.Wrapf(
		validate.ErrValidation,
		"cannot delete embedded rack %s",
		r.Name,
	)
}
