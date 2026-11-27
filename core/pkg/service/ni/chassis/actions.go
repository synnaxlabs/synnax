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
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/x/patch"
)

// Handle returns an error wrapping validate.ErrValidation, as only the Core creates a
// chassis.
func (CreatePayload) Handle(Chassis) (Chassis, error) {
	return Chassis{}, actions.CoreOnlyError("an NI chassis create")
}

// Handle applies the set to the chassis.
func (p SetPayload) Handle(c Chassis) (Chassis, error) {
	return apply(c, patch.Set(p))
}

// Handle applies the remove to the chassis.
func (p RemovePayload) Handle(c Chassis) (Chassis, error) {
	return apply(c, patch.Remove(p))
}

// Handle applies the insert to the chassis.
func (p InsertPayload) Handle(c Chassis) (Chassis, error) {
	return apply(c, patch.Insert(p))
}

// Handle applies the move to the chassis.
func (p MovePayload) Handle(c Chassis) (Chassis, error) {
	return apply(c, patch.Move(p))
}

func apply[A patch.Action](c Chassis, a A) (Chassis, error) {
	p, err := patch.Begin(a)
	if err == nil {
		err = c.Patch(p)
	}
	if err != nil {
		return Chassis{}, err
	}
	return c, nil
}
