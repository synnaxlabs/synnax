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
	"github.com/vmihailenco/msgpack/v5"
)

var (
	_ msgpack.CustomEncoder = Update{}
	_ msgpack.CustomDecoder = (*Update)(nil)
)

// EncodeMsgpack implements msgpack.CustomEncoder. It writes the active variant's fields
// with its group under "group", the same shape as the JSON encoding.
func (u Update) EncodeMsgpack(enc *msgpack.Encoder) error {
	switch v := u.Variant.(type) {
	case nil:
		return enc.EncodeNil()
	case IdentityUpdate:
		return enc.Encode(struct {
			Group UpdateType `msgpack:"group"`
			Identity
		}{Group: IdentityUpdateType, Identity: v.Identity})
	case ConfigurationUpdate:
		return enc.Encode(struct {
			Group UpdateType `msgpack:"group"`
			Configuration
		}{Group: ConfigurationUpdateType, Configuration: v.Configuration})
	default:
		return errors.Newf("Update: unknown variant %T", v)
	}
}

// DecodeMsgpack implements msgpack.CustomDecoder. It returns a validation error for a
// group no variant has.
func (u *Update) DecodeMsgpack(dec *msgpack.Decoder) error {
	raw, err := dec.DecodeRaw()
	if err != nil {
		return err
	}
	var disc struct {
		Group UpdateType `msgpack:"group"`
	}
	if err = msgpack.Unmarshal(raw, &disc); err != nil {
		return err
	}
	switch disc.Group {
	case IdentityUpdateType:
		var identity Identity
		if err = msgpack.Unmarshal(raw, &identity); err != nil {
			return err
		}
		u.Variant = IdentityUpdate{Identity: identity}
	case ConfigurationUpdateType:
		var configuration Configuration
		if err = msgpack.Unmarshal(raw, &configuration); err != nil {
			return err
		}
		u.Variant = ConfigurationUpdate{Configuration: configuration}
	default:
		return errors.Wrapf(
			validate.ErrValidation,
			"unknown update group %q",
			disc.Group,
		)
	}
	return nil
}

// Apply writes the field group that update carries to the device stored under key and
// keeps the other group as stored. It returns an error wrapping query.ErrNotFound when
// no device is stored under key and a validation error when update is invalid.
func (w Writer) Apply(ctx context.Context, key Key, update Update) error {
	switch v := update.Variant.(type) {
	case IdentityUpdate:
		if err := v.Validate(); err != nil {
			return err
		}
		return w.Update(ctx, key, func(d Device) (Device, error) {
			d.Rack = v.Rack
			d.Serial = v.Serial
			d.Model = v.Model
			d.Alias = v.Alias
			d.Simulated = v.Simulated
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
