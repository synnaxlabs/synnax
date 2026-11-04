// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package device

import (
	"context"

	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
)

// Writer is used to create, update, and delete devices within Synnax. The writer
// executes all operations within the transaction provided to the Service.NewWriter
// method. If no transaction is provided, the writer will execute operations directly on
// the database.
type Writer struct {
	svc      *Service
	tx       gorp.Tx
	otg      ontology.Writer
	status   status.Writer
	statuses *status.Service
	table    *gorp.Table[Key, Device]
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

func resolveStatus(d *Device, provided *Status) *Status {
	if provided == nil {
		return &Status{
			Owner:   new(d.ResourceID()),
			Name:    d.Name,
			Time:    telem.Now(),
			Variant: status.VariantWarning,
			Message: "Device status unknown",
			Details: StatusDetails{Rack: d.Rack},
		}
	}
	stat := *provided
	stat.Owner = new(d.ResourceID())
	stat.Name = d.Name
	stat.Details.Rack = d.Rack
	return &stat
}

// writeStatus sets d.Status to the status that d now has in the DB. A provided status
// replaces the stored row only if overwritable is true. The default status is written
// only if no row exists, because a stored row may be a live status from the Driver.
func (w Writer) writeStatus(
	ctx context.Context,
	d *Device,
	provided *Status,
	overwritable bool,
) error {
	if provided != nil && overwritable {
		d.Status = resolveStatus(d, provided)
		return w.status.Set(ctx, d.Status)
	}
	var stored Status
	err := w.statuses.NewRetrieve[StatusDetails]().
		Where(status.MatchOwners[StatusDetails](d.ResourceID())).
		Entry(&stored).
		Exec(ctx, w.tx)
	if errors.Is(err, query.ErrNotFound) {
		d.Status = resolveStatus(d, provided)
		return w.status.Set(ctx, d.Status)
	}
	if err != nil {
		return err
	}
	d.Status = resolveStatus(d, &stored)
	if stored.Name == d.Name && stored.Details.Rack == d.Rack {
		return nil
	}
	return w.status.Set(ctx, d.Status)
}

// Create creates or updates the given device. An existing device keeps its group, and
// its parent when sent with none, unless it moves to another rack. A status provided on
// the device replaces the stored status if the device is new, renamed, or moved to
// another rack or parent. With no provided status, the stored status stays, and only a
// device with no stored status gets the default "unknown" status. Create sets
// device.Status to the resulting status.
func (w Writer) Create(ctx context.Context, device *Device) error {
	return w.inTx(ctx, func(w Writer) error { return w.create(ctx, device) })
}

func (w Writer) create(ctx context.Context, device *Device) error {
	if err := device.Validate(); err != nil {
		return err
	}
	providedStatus := device.Status
	device.Status = nil // Status stored separately, not in gorp
	var existing Device
	err := w.table.
		NewRetrieve().
		Where(gorp.MatchKeys[Key, Device](device.Key)).
		Entry(&existing).
		Exec(ctx, w.tx)
	if errors.Skip(err, query.ErrNotFound) != nil {
		return err
	}
	exists := err == nil
	if exists {
		device.Group = existing.Group
		if device.Rack != existing.Rack {
			device.Group = nil
		} else if device.Parent == nil {
			device.Parent = existing.Parent
		}
	}
	if err = w.checkGroup(ctx, device.Group, *device); err != nil {
		return err
	}
	if err = w.table.
		NewCreate().
		Entry(device).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	unmoved := exists &&
		device.Rack == existing.Rack &&
		lo.FromPtr(device.Parent) == lo.FromPtr(existing.Parent)
	overwritable := !unmoved || device.Name != existing.Name
	if err = w.writeStatus(ctx, device, providedStatus, overwritable); err != nil {
		return err
	}
	if exists {
		return nil
	}
	return w.otg.DefineResources(ctx, device.ResourceID())
}

// CreateMany creates or updates the given devices. If devices with the same key already
// exist, they will be overwritten.
func (w Writer) CreateMany(ctx context.Context, devices *[]Device) error {
	for i := range *devices {
		if err := w.Create(ctx, &(*devices)[i]); err != nil {
			return err
		}
	}
	return nil
}

// SetGroup moves the devices with keys into g, or out of their group when g is nil. It
// returns a validation error when g does not sit under a device's rack.
func (w Writer) SetGroup(ctx context.Context, g *group.Key, keys ...Key) error {
	return setGroup(w.table, g, keys...).
		ChangeErr(func(ctx gorp.Context, d Device) (Device, error) {
			return d, w.checkGroup(ctx, g, d)
		}).
		Exec(ctx, w.tx)
}

func (w Writer) checkGroup(ctx context.Context, key *group.Key, d Device) error {
	return group.CheckMember(
		ctx,
		w.tx,
		key,
		resource.TypeDevice,
		new(d.Rack.ResourceID()),
	)
}

// Rename renames the device with the given key and its status, keeping every other
// field. It returns query.ErrNotFound if no device with the key exists and a validation
// error if name is empty.
func (w Writer) Rename(ctx context.Context, key Key, name string) error {
	if err := w.table.NewUpdate().
		Where(gorp.MatchKeys[Key, Device](key)).
		ChangeErr(func(_ gorp.Context, d Device) (Device, error) {
			d.Name = name
			if err := d.Validate(); err != nil {
				return Device{}, err
			}
			return d, nil
		}).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	// A missing status row is valid: the next Create heals it.
	return errors.Skip(
		w.status.RenameOwned(ctx, ResourceID(key), name),
		query.ErrNotFound,
	)
}

// Delete deletes the device with the given key, its status, and every device it holds.
func (w Writer) Delete(ctx context.Context, key Key) error {
	if err := w.otg.DeleteResources(ctx, ResourceID(key)); err != nil {
		return err
	}
	return w.table.NewDelete().Where(gorp.MatchKeys[Key, Device](key)).Exec(ctx, w.tx)
}
