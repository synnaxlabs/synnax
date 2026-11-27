// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package chassis

import (
	"context"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// Apply replaces the field group that update carries on the chassis stored under key
// and keeps the other group as stored. It returns an error wrapping query.ErrNotFound
// when no chassis is stored under key and a validation error when update is invalid. A
// transaction that commits between the read and the write of this one is overwritten.
func (w Writer) Apply(ctx context.Context, key Key, update Update) error {
	switch v := update.Variant.(type) {
	case IdentityUpdate:
		if err := v.Validate(); err != nil {
			return err
		}
		return w.Update(ctx, key, func(c Chassis) (Chassis, error) {
			c.Rack = v.Rack
			c.Serial = v.Serial
			c.ResourceName = v.ResourceName
			c.Model = v.Model
			return c, nil
		})
	case ConfigurationUpdate:
		if err := v.Validate(); err != nil {
			return err
		}
		return w.Update(ctx, key, func(c Chassis) (Chassis, error) {
			c.Name = v.Name
			return c, nil
		})
	default:
		return errors.Wrapf(
			validate.ErrValidation,
			"chassis update has no field group",
		)
	}
}
