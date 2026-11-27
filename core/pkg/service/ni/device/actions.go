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
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/x/patch"
)

// Handle returns an error wrapping validate.ErrValidation, as only the Core creates a
// device.
func (CreatePayload) Handle(Device) (Device, error) {
	return Device{}, actions.CoreOnlyError("an NI device create")
}

// Handle applies the set to the device.
func (p SetPayload) Handle(d Device) (Device, error) {
	return apply(d, patch.Set(p))
}

// Handle applies the remove to the device.
func (p RemovePayload) Handle(d Device) (Device, error) {
	return apply(d, patch.Remove(p))
}

// Handle applies the insert to the device.
func (p InsertPayload) Handle(d Device) (Device, error) {
	return apply(d, patch.Insert(p))
}

// Handle applies the move to the device.
func (p MovePayload) Handle(d Device) (Device, error) {
	return apply(d, patch.Move(p))
}

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
