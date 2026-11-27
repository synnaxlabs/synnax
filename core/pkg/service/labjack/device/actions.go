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

	"github.com/synnaxlabs/x/patch"
)

// Handle applies the set to the device.
func (p SetPayload) Handle(d Device) (Device, error) { return apply(d, patch.Set(p)) }

// Handle applies the remove to the device.
func (p RemovePayload) Handle(d Device) (Device, error) {
	return apply(d, patch.Remove(p))
}

// Handle applies the insert to the device.
func (p InsertPayload) Handle(d Device) (Device, error) {
	return apply(d, patch.Insert(p))
}

// Handle applies the move to the device.
func (p MovePayload) Handle(d Device) (Device, error) { return apply(d, patch.Move(p)) }

func apply[A patch.Action](d Device, a A) (Device, error) {
	c, err := patch.Begin(a)
	if err == nil {
		err = d.Patch(c)
	}
	if err != nil {
		return Device{}, err
	}
	return d, nil
}

// Apply reduces acts onto the device stored under key, stores the result, and attaches
// acts to the transaction of w, so its commit broadcasts them with the dispatch key of
// w. The scanner sets only the identity paths: rack, serial, connection, and model.
// Apply returns an error wrapping query.ErrNotFound when no device is stored under key,
// and a validation error when an action fails or the result is invalid, changes the
// model, carries ports for another model, gives a pin two roles, or maps a Mux80 port
// or address line against the Mux80 state.
func (w Writer) Apply(ctx context.Context, key Key, acts ...Action) error {
	if err := w.update(ctx, key, func(d Device) (Device, error) {
		return Reduce(d, acts...)
	}); err != nil {
		return err
	}
	w.dispatcher.Attach(w.tx, key, acts...)
	return nil
}
