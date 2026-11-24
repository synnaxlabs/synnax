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
	"maps"
	"slices"
	"strconv"
	"strings"
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
			Details: StatusDetails{Device: d.Key},
		}
	} else if err != nil {
		return err
	}
	s.Name = d.Name
	s.Details.Rack = d.Rack
	return w.status.Set(ctx, &s)
}

// firstMux80Port is the lowest analog input port that a Mux80 adds to a T7.
const firstMux80Port = 48

// validateDevice rejects an invalid device, one whose ports are for another model, and
// a T7 that maps a Mux80 port while no Mux80 is attached.
func validateDevice(d Device) error {
	if err := d.Validate(); err != nil {
		return err
	}
	if err := d.ValidatePorts(); err != nil {
		return err
	}
	v := validate.New("device")
	if t7, ok := d.Ports.Variant.(T7Ports); ok && !t7.Mux80Attached {
		for _, port := range slices.Sorted(maps.Keys(t7.AnalogInput)) {
			n, err := strconv.Atoi(strings.TrimPrefix(string(port), "AIN"))
			v.Ternaryf(
				"ports.analog_input",
				err == nil && n >= firstMux80Port,
				"%s needs mux80_attached",
				port,
			)
		}
	}
	return v.Error()
}

// Create stores d and writes an unknown status when it has none. An existing status
// takes the name and rack of d. A device with a nil key gets a new one. It returns a
// validation error when d is invalid, its ports are for another model, it maps a Mux80
// port while no Mux80 is attached, it replaces a device of another model, or its rack
// does not exist.
func (w Writer) Create(ctx context.Context, d *Device) error {
	if err := validateDevice(*d); err != nil {
		return err
	}
	if d.Key == uuid.Nil() {
		d.Key = uuid.New()
	} else if err := w.refuseReplacedModel(ctx, *d); err != nil {
		return err
	}
	if err := w.table.NewCreate().Entry(d).Exec(ctx, w.tx); err != nil {
		return err
	}
	return w.setStatus(ctx, *d)
}

// update applies change to the device stored under key and stores the result. A rename
// or rack move updates the status, or writes it as unknown when it is missing.
func (w Writer) update(
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
			if err = refuseModelChange(d.Model, updated.Model); err != nil {
				return Device{}, err
			}
			updated.Key = key
			if err = validateDevice(updated); err != nil {
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

func (w Writer) refuseReplacedModel(ctx context.Context, d Device) error {
	var stored Device
	err := w.table.NewRetrieve().
		Where(gorp.MatchKeys[Key, Device](d.Key)).
		Entry(&stored).
		Exec(ctx, w.tx)
	if errors.Is(err, query.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	return refuseModelChange(stored.Model, d.Model)
}

func refuseModelChange(stored, next Model) error {
	v := validate.New("device")
	v.Ternaryf("model", next != stored, "cannot change from %s to %s", stored, next)
	return v.Error()
}

// Delete removes the devices stored under keys with their statuses. Delete is
// idempotent.
func (w Writer) Delete(ctx context.Context, keys ...Key) error {
	return w.table.NewDelete().
		Where(gorp.MatchKeys[Key, Device](keys...)).
		Exec(ctx, w.tx)
}
