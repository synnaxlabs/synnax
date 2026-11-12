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
	"uuid"

	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
)

// Writer creates, updates, and deletes NI devices in the transaction given to
// Service.NewWriter.
type Writer struct {
	tx       gorp.Tx
	status   status.Writer
	statuses *status.Service
	table    *gorp.Table[Key, Device]
}

func (w Writer) setStatus(ctx context.Context, d Device) error {
	var s Status
	err := w.statuses.NewRetrieve[StatusDetails]().
		Where(status.MatchOwners[StatusDetails](d.ResourceID())).
		Entry(&s).
		Exec(ctx, w.tx)
	if errors.Is(err, query.ErrNotFound) {
		s = Status{
			Owner:   new(d.ResourceID()),
			Time:    telem.Now(),
			Variant: status.VariantWarning,
			Message: "Status unknown",
			Details: StatusDetails{Device: d.Key},
		}
	} else if err != nil {
		return err
	}
	s.Name = d.Name
	s.Details.Rack = d.Rack
	return w.status.Set(ctx, &s)
}

// Create stores d and writes an unknown status when it has none. An existing status
// takes the name and rack of d. A device with a nil key gets a new one. It returns a
// validation error when d is invalid or its rack does not exist.
func (w Writer) Create(ctx context.Context, d *Device) error {
	if d.Key == uuid.Nil() {
		d.Key = uuid.New()
	}
	if err := d.Validate(); err != nil {
		return err
	}
	if err := w.table.NewCreate().Entry(d).Exec(ctx, w.tx); err != nil {
		return err
	}
	return w.setStatus(ctx, *d)
}

// Update applies change to the device stored under key and stores the result, so a
// caller changes one field group and keeps the other as stored. A rename or rack move
// updates the status, or writes it as unknown when it is missing. It returns an error
// wrapping query.ErrNotFound when no device is stored under key and a validation error
// when the result is invalid. A transaction that commits between the read and the
// write of this one is overwritten.
func (w Writer) Update(
	ctx context.Context,
	key Key,
	change func(Device) (Device, error),
) error {
	var prev, next Device
	if err := w.table.NewUpdate().
		Where(gorp.MatchKeys[Key, Device](key)).
		ChangeErr(func(_ gorp.Context, d Device) (Device, error) {
			prev = d
			updated, err := change(d)
			if err != nil {
				return Device{}, err
			}
			updated.Key = key
			if err = updated.Validate(); err != nil {
				return Device{}, err
			}
			next = updated
			return updated, nil
		}).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	if next.Name != prev.Name || next.Rack != prev.Rack {
		return w.setStatus(ctx, next)
	}
	return nil
}

// Delete removes the devices stored under keys with their statuses. Delete is
// idempotent.
func (w Writer) Delete(ctx context.Context, keys ...Key) error {
	return w.table.NewDelete().
		Where(gorp.MatchKeys[Key, Device](keys...)).
		Exec(ctx, w.tx)
}
