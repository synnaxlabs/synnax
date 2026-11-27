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

// keys returns the port names of m in order.
func keys[K ~string, V any](m map[K]V) []string {
	names := make([]string, 0, len(m))
	for k := range m {
		names = append(names, string(k))
	}
	slices.Sort(names)
	return names
}

// refuseSharedPins rejects a digital port that is both an input and an output, and on a
// T4 an analog input whose pin is also mapped as a digital port. The T4's AIN4 to
// AIN11 are the pins of DIO4 to DIO11.
func refuseSharedPins(v *validate.Validator, analog, input, output []string) {
	for _, port := range output {
		v.Ternaryf(
			"ports.digital_output",
			slices.Contains(input, port),
			"%s is also a digital input",
			port,
		)
	}
	for _, port := range analog {
		dio := "DIO" + strings.TrimPrefix(port, "AIN")
		v.Ternaryf(
			"ports.analog_input",
			slices.Contains(input, dio) || slices.Contains(output, dio),
			"%s shares its pin with %s",
			port,
			dio,
		)
	}
}

// mux80AddressLines are the T7 pins that address an attached Mux80.
var mux80AddressLines = []T7DigitalPort{
	T7DigitalPortDio20,
	T7DigitalPortDio21,
	T7DigitalPortDio22,
}

// refuseMux80AddressLines rejects a digital port of p that addresses its Mux80.
func refuseMux80AddressLines(v *validate.Validator, p T7Ports) {
	for _, port := range mux80AddressLines {
		_, in := p.DigitalInput[port]
		_, out := p.DigitalOutput[port]
		v.Ternaryf(
			"ports.digital_input",
			in,
			"%s addresses the attached Mux80",
			port,
		)
		v.Ternaryf(
			"ports.digital_output",
			out,
			"%s addresses the attached Mux80",
			port,
		)
	}
}

// validateDevice rejects an invalid device, one whose ports are for another model, one
// that gives a pin two roles, a T7 that maps a Mux80 port while no Mux80 is attached,
// and a T7 that maps a Mux80 address line while one is attached.
func validateDevice(d Device) error {
	if err := d.Validate(); err != nil {
		return err
	}
	if err := d.ValidatePorts(); err != nil {
		return err
	}
	v := validate.New("device")
	switch p := d.Ports.Variant.(type) {
	case T4Ports:
		refuseSharedPins(
			v,
			keys(p.AnalogInput),
			keys(p.DigitalInput),
			keys(p.DigitalOutput),
		)
	case T7Ports:
		refuseSharedPins(v, nil, keys(p.DigitalInput), keys(p.DigitalOutput))
	case T8Ports:
		refuseSharedPins(v, nil, keys(p.DigitalInput), keys(p.DigitalOutput))
	}
	t7, ok := d.Ports.Variant.(T7Ports)
	switch {
	case ok && t7.Mux80Attached:
		refuseMux80AddressLines(v, t7)
	case ok:
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
// validation error when d is invalid, its ports are for another model, it gives a pin
// two roles, it maps a Mux80 port or address line against the Mux80 state, it replaces
// a device of another model, or its rack does not exist.
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
