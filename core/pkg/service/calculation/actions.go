// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package calculation

import (
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/x/patch"
)

// Handle returns an error wrapping validate.ErrValidation, as only the Core creates a
// calculation.
func (CreatePayload) Handle(Calculation) (Calculation, error) {
	return Calculation{}, actions.CoreOnlyError("a calculation create")
}

// Handle applies the set to the calculation.
func (p SetPayload) Handle(c Calculation) (Calculation, error) {
	return apply(c, patch.Set(p))
}

// Handle applies the remove to the calculation.
func (p RemovePayload) Handle(c Calculation) (Calculation, error) {
	return apply(c, patch.Remove(p))
}

// Handle applies the insert to the calculation.
func (p InsertPayload) Handle(c Calculation) (Calculation, error) {
	return apply(c, patch.Insert(p))
}

// Handle applies the move to the calculation.
func (p MovePayload) Handle(c Calculation) (Calculation, error) {
	return apply(c, patch.Move(p))
}

func apply[A patch.Action](c Calculation, a A) (Calculation, error) {
	ch, err := patch.Begin(a)
	if err == nil {
		err = c.Patch(ch)
	}
	if err != nil {
		return Calculation{}, err
	}
	return c, nil
}
