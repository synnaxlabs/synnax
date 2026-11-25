// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package slave

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

// Apply replaces the field group that update carries on the slave stored under key
// and keeps the other group as stored. It returns an error wrapping query.ErrNotFound
// when no slave is stored under key, and a validation error when update is invalid. A
// rename or rack move updates the status, or writes it as unknown when it is missing.
func (w Writer) Apply(ctx context.Context, key Key, update Update) error {
	switch v := update.Variant.(type) {
	case IdentityUpdate:
		if err := v.Validate(); err != nil {
			return err
		}
		return w.update(ctx, key, func(s Slave) (Slave, error) {
			s.Rack = v.Rack
			s.Serial = v.Serial
			s.VendorID = v.VendorID
			s.ProductCode = v.ProductCode
			s.Revision = v.Revision
			s.Master = v.Master
			s.Pdos = v.Pdos
			s.OpModes = v.OpModes
			s.SwitchedAt = v.SwitchedAt
			return s, nil
		})
	case ConfigurationUpdate:
		if err := v.Validate(); err != nil {
			return err
		}
		return w.update(ctx, key, func(s Slave) (Slave, error) {
			s.Name = v.Name
			s.Identifier = v.Identifier
			s.Disabled = v.Disabled
			s.OpMode = v.OpMode
			s.StartupParameters = v.StartupParameters
			s.ReadIndex = v.ReadIndex
			s.WriteStateIndex = v.WriteStateIndex
			s.ReadBindings = v.ReadBindings
			s.WriteBindings = v.WriteBindings
			return s, nil
		})
	default:
		return errors.Wrapf(
			validate.ErrValidation,
			"slave update has no field group",
		)
	}
}
