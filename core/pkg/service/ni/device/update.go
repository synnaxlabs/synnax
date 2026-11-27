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

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// Apply replaces the field group that update carries on the device stored under key and
// keeps the other group as stored. It returns an error wrapping query.ErrNotFound when
// no device is stored under key and a validation error when update is invalid. A
// transaction that commits between the read and the write of this one is overwritten.
func (w Writer) Apply(ctx context.Context, key Key, update Update) error {
	switch v := update.Variant.(type) {
	case IdentityUpdate:
		if err := v.Validate(); err != nil {
			return err
		}
		return w.Update(ctx, key, func(d Device) (Device, error) {
			d.Rack = v.Rack
			d.Serial = v.Serial
			d.ResourceName = v.ResourceName
			d.Model = v.Model
			d.Chassis = v.Chassis
			return d, nil
		})
	case ConfigurationUpdate:
		if err := v.Validate(); err != nil {
			return err
		}
		return w.Update(ctx, key, func(d Device) (Device, error) {
			d.Name = v.Name
			d.Identifier = v.Identifier
			d.AnalogInput = v.AnalogInput
			d.AnalogOutput = v.AnalogOutput
			d.CounterInput = v.CounterInput
			d.DigitalInput = v.DigitalInput
			d.DigitalOutput = v.DigitalOutput
			return d, nil
		})
	default:
		return errors.Wrapf(
			validate.ErrValidation,
			"device update has no field group",
		)
	}
}
