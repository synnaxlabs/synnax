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
	_ msgpack.CustomEncoder = Ports{}
	_ msgpack.CustomDecoder = (*Ports)(nil)
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

// EncodeMsgpack implements msgpack.CustomEncoder. It writes the active variant's fields
// with its model under "model", the same shape as the JSON encoding.
func (p Ports) EncodeMsgpack(enc *msgpack.Encoder) error {
	switch v := p.Variant.(type) {
	case nil:
		return enc.EncodeNil()
	case T4Ports:
		return enc.Encode(struct {
			Model Model `msgpack:"model"`
			T4Ports
		}{Model: ModelT4, T4Ports: v})
	case T7Ports:
		return enc.Encode(struct {
			Model Model `msgpack:"model"`
			T7Ports
		}{Model: ModelT7, T7Ports: v})
	case T8Ports:
		return enc.Encode(struct {
			Model Model `msgpack:"model"`
			T8Ports
		}{Model: ModelT8, T8Ports: v})
	default:
		return errors.Newf("Ports: unknown variant %T", v)
	}
}

// DecodeMsgpack implements msgpack.CustomDecoder. It returns a validation error for a
// model no variant has.
func (p *Ports) DecodeMsgpack(dec *msgpack.Decoder) error {
	raw, err := dec.DecodeRaw()
	if err != nil {
		return err
	}
	var disc struct {
		Model Model `msgpack:"model"`
	}
	if err = msgpack.Unmarshal(raw, &disc); err != nil {
		return err
	}
	switch disc.Model {
	case ModelT4:
		var v T4Ports
		err = msgpack.Unmarshal(raw, &v)
		p.Variant = v
	case ModelT7:
		var v T7Ports
		err = msgpack.Unmarshal(raw, &v)
		p.Variant = v
	case ModelT8:
		var v T8Ports
		err = msgpack.Unmarshal(raw, &v)
		p.Variant = v
	default:
		return errors.Wrapf(validate.ErrValidation, "unknown model %q", disc.Model)
	}
	return err
}

// withPorts returns the device of the model of ports, with base and the configuration
// that ports holds.
func withPorts(b Base, ports Ports) Device {
	switch p := ports.Variant.(type) {
	case T4Ports:
		return Device{Variant: T4Device{Base: b}}
	case T7Ports:
		return Device{Variant: T7Device{Base: b, Mux80Attached: p.Mux80Attached}}
	case T8Ports:
		return Device{Variant: T8Device{Base: b}}
	default:
		return Device{}
	}
}

// Apply replaces the field group that update carries on the device stored under key and
// keeps the other group as stored. It returns an error wrapping query.ErrNotFound when
// no device is stored under key and a validation error when update is invalid or names
// a different model than the stored device. A transaction that commits between the
// read and the write of this one is overwritten.
func (w Writer) Apply(ctx context.Context, key Key, update Update) error {
	switch v := update.Variant.(type) {
	case IdentityUpdate:
		if err := v.Validate(); err != nil {
			return err
		}
		return w.update(ctx, key, func(d Device) (Device, error) {
			if err := refuseModelChange(d.Model(), v.Model); err != nil {
				return Device{}, err
			}
			b := d.Base()
			b.Rack = v.Rack
			b.Serial = v.Serial
			b.Connection = v.Connection
			d.SetBase(b)
			return d, nil
		})
	case ConfigurationUpdate:
		if err := v.Validate(); err != nil {
			return err
		}
		return w.update(ctx, key, func(d Device) (Device, error) {
			b := d.Base()
			b.Name = v.Name
			b.Identifier = v.Identifier
			b.ReadIndex = v.ReadIndex
			b.WriteStateIndex = v.WriteStateIndex
			return withPorts(b, v.Ports), nil
		})
	default:
		return errors.Wrapf(validate.ErrValidation, "device update has no field group")
	}
}
