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
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
)

// Writer is used to create, update, and delete devices within Synnax. The writer
// executes all operations within the transaction provided to the Service.NewWriter
// method.
type Writer struct {
	tx         gorp.Tx
	status     status.Writer
	statuses   *status.Service
	table      *gorp.Table[Key, Device]
	dispatcher actions.Dispatcher[Key, Action]
}

// WithDispatchKey returns a copy of the writer that stamps dispatchKey onto the
// broadcasts it emits, so the client that sent the write recognizes its own echo.
func (w Writer) WithDispatchKey(dispatchKey string) Writer {
	w.dispatcher = w.dispatcher.WithDispatchKey(dispatchKey)
	return w
}

// resolveStatus returns the status to persist for d. The owner selects the status, so a
// provided key is ignored.
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
	stat.Key = ""
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

// Create creates or updates the given device and attaches a create action that holds
// it. Create writes the parent as sent, so an update must carry the current parent. A
// new device with a parent cannot sit in a group. An existing device keeps its group
// unless it moves to another rack or has a parent. A status provided on the device
// replaces the stored status if the device is new, renamed, or moved to another rack or
// parent. With no provided status, the stored status stays, and only a device with no
// stored status gets the default "unknown" status. Create sets device.Status to the
// resulting status.
func (w Writer) Create(ctx context.Context, device *Device) error {
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
		if device.Rack != existing.Rack || device.Parent != nil {
			device.Group = nil
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
	w.dispatcher.Attach(
		w.tx,
		device.Key,
		NewCreateAction(CreatePayload{Device: *device}),
	)
	return nil
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

// SetGroup applies a set of the group to each device with keys, moving it into g, or
// out of its group when g is nil. It returns a validation error when g does not sit
// under a device's rack, or when a device has a parent.
func (w Writer) SetGroup(ctx context.Context, g *group.Key, keys ...Key) error {
	set := NewSetAction(SetPayload{Path: []string{"group"}, Value: g})
	for _, k := range keys {
		if err := w.Apply(ctx, k, set); err != nil {
			return err
		}
	}
	return nil
}

func (w Writer) checkGroup(ctx context.Context, key *group.Key, d Device) error {
	holder := d.Rack.ResourceID()
	if d.Parent != nil {
		holder = ResourceID(*d.Parent)
	}
	return group.CheckMember(ctx, w.tx, key, resource.TypeDevice, &holder)
}

// Rename applies a set of the name to the device with the given key. It returns
// query.ErrNotFound if no device with the key exists and a validation error if name is
// empty.
func (w Writer) Rename(ctx context.Context, key Key, name string) error {
	return w.Apply(
		ctx,
		key,
		NewSetAction(SetPayload{Path: []string{"name"}, Value: name}),
	)
}

// Apply applies the actions to the device with the given key and attaches them to the
// transaction of w. A change of name or rack also updates the device's status. Apply
// returns an error wrapping validate.ErrValidation when an action is a create, the
// result fails to validate, or the device's group does not sit under its rack or
// parent, and an error wrapping query.ErrNotFound when no device has the key.
func (w Writer) Apply(ctx context.Context, key Key, acts ...Action) error {
	var prev Device
	if err := w.table.NewRetrieve().
		Where(gorp.MatchKeys[Key, Device](key)).
		Entry(&prev).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	next, err := Reduce(prev, acts...)
	if err != nil {
		return err
	}
	if err = next.Validate(); err != nil {
		return err
	}
	if err = w.checkGroup(ctx, next.Group, next); err != nil {
		return err
	}
	if err = w.table.NewCreate().Entry(&next).Exec(ctx, w.tx); err != nil {
		return err
	}
	if next.Name != prev.Name || next.Rack != prev.Rack {
		if err = w.writeStatus(ctx, &next, nil, false); err != nil {
			return err
		}
	}
	w.dispatcher.Attach(w.tx, key, acts...)
	return nil
}

// Delete deletes the device with the given key, its status, and every device it holds.
func (w Writer) Delete(ctx context.Context, key Key) error {
	return w.table.NewDelete().Where(gorp.MatchKeys[Key, Device](key)).Exec(ctx, w.tx)
}
