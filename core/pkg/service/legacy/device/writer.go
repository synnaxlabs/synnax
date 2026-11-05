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

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
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
	tx     gorp.Tx
	otg    ontology.Writer
	group  group.Group
	status status.Writer
	table  *gorp.Table[Key, Device]
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
	key, err := w.status.KeyOf(ctx, d.ResourceID())
	if errors.Is(err, query.ErrNotFound) {
		d.Status = resolveStatus(d, provided)
		return w.status.Set(ctx, d.Status)
	}
	if err != nil {
		return err
	}
	var stored Status
	if err = gorp.NewRetrieve[string, Status]().
		Where(gorp.MatchKeys[string, Status](key)).
		Entry(&stored).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	d.Status = resolveStatus(d, &stored)
	if stored.Name == d.Name && stored.Details.Rack == d.Rack {
		return nil
	}
	return w.status.Set(ctx, d.Status)
}

// Create creates or updates the given device. If device.Parent is non-zero, the device
// is parented to that ontology resource; otherwise it defaults to the device's rack.
// A status provided on the device replaces the stored status if the device is new,
// renamed, moved to another rack, or sent with a parent. With no provided status, the
// stored status stays, and only a device with no stored status gets the default
// "unknown" status. Create sets device.Status to the resulting status.
func (w Writer) Create(ctx context.Context, device *Device) error {
	if err := device.Validate(); err != nil {
		return err
	}
	providedStatus := device.Status
	device.Status = nil // Status stored separately, not in gorp

	callerProvidedParent := device.Parent != nil && !device.Parent.IsZero()
	if !callerProvidedParent {
		rackID := device.Rack.ResourceID()
		device.Parent = &rackID
	}
	parentID := *device.Parent
	device.Parent = nil // Parent is not stored in gorp

	var existing Device
	err := w.table.
		NewRetrieve().
		Where(gorp.MatchKeys[Key, Device](device.Key)).
		Entry(&existing).
		Exec(ctx, w.tx)
	isNotFound := errors.Is(err, query.ErrNotFound)
	if err != nil && !isNotFound {
		return err
	}
	exists := !isNotFound
	if err = w.table.
		NewCreate().
		Entry(device).
		Exec(ctx, w.tx); err != nil {
		return err
	}

	device.Parent = &parentID

	// If the device already exists and its rack hasn't changed and no explicit
	// parent was provided, skip redefining the ontology relationship.
	unmoved := exists && device.Rack == existing.Rack && !callerProvidedParent
	overwritable := !unmoved || device.Name != existing.Name
	if err = w.writeStatus(ctx, device, providedStatus, overwritable); err != nil {
		return err
	}
	if unmoved {
		return nil
	}
	otgID := device.ResourceID()
	if err = w.otg.DefineResources(ctx, otgID); err != nil {
		return err
	}
	if err = w.otg.DeleteIncomingRelationshipsOfType(
		ctx,
		otgID,
		ontology.RelationshipTypeParentOf,
	); err != nil {
		return err
	}
	return w.otg.DefineRelationships(
		ctx,
		parentID,
		ontology.RelationshipTypeParentOf,
		otgID,
	)
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
	statusKey, err := w.status.KeyOf(ctx, ResourceID(key))
	if err == nil {
		err = w.status.Rename(ctx, statusKey, name)
	}
	return errors.Skip(err, query.ErrNotFound)
}

// Delete deletes the device with the given key and its associated status.
func (w Writer) Delete(ctx context.Context, key Key) error {
	if err := w.otg.DeleteResources(ctx, ResourceID(key)); err != nil {
		return err
	}
	return w.table.NewDelete().Where(gorp.MatchKeys[Key, Device](key)).Exec(ctx, w.tx)
}
