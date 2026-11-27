// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package server

import (
	"context"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/validate"
)

// ApplyUpdate replaces the field group that update carries on the server stored under
// key and keeps the other group as stored. It applies the group as one set action per
// field, so subscribers to the action channel see the write. It returns an error
// wrapping query.ErrNotFound when no server is stored under key and a validation error
// when update is invalid.
func (w Writer) ApplyUpdate(ctx context.Context, key Key, update Update) error {
	acts, err := update.actions()
	if err != nil {
		return err
	}
	return w.Apply(ctx, key, acts...)
}

// actions returns one set action for each field of the group that u replaces.
func (u Update) actions() ([]Action, error) {
	set := func(path string, value any) Action {
		return NewSetAction(SetPayload{Path: []string{path}, Value: value})
	}
	switch v := u.Variant.(type) {
	case IdentityUpdate:
		return []Action{
			set("rack", v.Rack),
			set("name", v.Name),
			set("connection", v.Connection),
		}, nil
	case ConfigurationUpdate:
		return []Action{set("reads", v.Reads), set("writes", v.Writes)}, nil
	default:
		return nil, errors.Wrapf(
			validate.ErrValidation,
			"server update has no field group",
		)
	}
}
