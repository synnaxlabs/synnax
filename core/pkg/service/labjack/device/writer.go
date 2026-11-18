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
	"github.com/synnaxlabs/x/validate"
)

// Writer creates, updates, and deletes LabJack devices in the transaction given to
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
			Details: StatusDetails{Device: d.Base().Key},
		}
	} else if err != nil {
		return err
	}
	s.Name = d.Base().Name
	s.Details.Rack = d.Base().Rack
	return w.status.Set(ctx, &s)
}

// validateDevice rejects a device with no known model, which the generated Validate
// accepts.
func validateDevice(d Device) error {
	v := validate.New("device")
	v.Ternary("model", d.Model() == "", "required")
	if err := v.Error(); err != nil {
		return err
	}
	return d.Validate()
}

// Create stores d and writes an unknown status when it has none. An existing status
// takes the name and rack of d. A device with a nil key gets a new one. It returns a
// validation error when d has no model, is invalid, or its rack does not exist.
func (w Writer) Create(ctx context.Context, d *Device) error {
	if err := validateDevice(*d); err != nil {
		return err
	}
	if b := d.Base(); b.Key == uuid.Nil() {
		b.Key = uuid.New()
		d.SetBase(b)
	}
	if err := w.table.NewCreate().Entry(d).Exec(ctx, w.tx); err != nil {
		return err
	}
	return w.setStatus(ctx, *d)
}

// Configure writes the user's fields of d to the device stored under its key and keeps
// the stored rack, connection, serial, and model, which only the scanner writes. A
// rename updates the status, or writes it as unknown when it is missing. It returns a
// validation error when d is invalid or has a different model than the stored device,
// and an error wrapping query.ErrNotFound when no device is stored under its key. A
// device built from an earlier read overwrites every change committed since.
func (w Writer) Configure(ctx context.Context, d Device) error {
	v := validate.New("device")
	v.Ternary("model", d.Model() == "", "required")
	if err := v.Error(); err != nil {
		return err
	}
	var prev Device
	if err := w.table.NewUpdate().
		Where(gorp.MatchKeys[Key, Device](d.Base().Key)).
		ChangeErr(func(_ gorp.Context, stored Device) (Device, error) {
			v := validate.New("device")
			v.Ternaryf(
				"model",
				d.Model() != stored.Model(),
				"cannot change from %s to %s",
				stored.Model(),
				d.Model(),
			)
			if err := v.Error(); err != nil {
				return Device{}, err
			}
			prev = stored
			b, kept := d.Base(), stored.Base()
			b.Rack, b.Connection, b.Serial = kept.Rack, kept.Connection, kept.Serial
			d.SetBase(b)
			return d, validateDevice(d)
		}).
		Exec(ctx, w.tx); err != nil {
		return err
	}
	if d.Base().Name != prev.Base().Name {
		return w.setStatus(ctx, d)
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
