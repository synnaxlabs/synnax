// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package enumeration

import (
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/x/patch"
)

// Handle returns an error wrapping validate.ErrValidation, as only the Core creates an
// enumeration.
func (CreatePayload) Handle(Enumeration) (Enumeration, error) {
	return Enumeration{}, actions.CoreOnlyError("an enumeration create")
}

// Handle applies the set to the enumeration.
func (p SetPayload) Handle(e Enumeration) (Enumeration, error) {
	return apply(e, patch.Set(p))
}

// Handle applies the remove to the enumeration.
func (p RemovePayload) Handle(e Enumeration) (Enumeration, error) {
	return apply(e, patch.Remove(p))
}

// Handle applies the insert to the enumeration.
func (p InsertPayload) Handle(e Enumeration) (Enumeration, error) {
	return apply(e, patch.Insert(p))
}

// Handle applies the move to the enumeration.
func (p MovePayload) Handle(e Enumeration) (Enumeration, error) {
	return apply(e, patch.Move(p))
}

func apply[A patch.Action](e Enumeration, a A) (Enumeration, error) {
	c, err := patch.Begin(a)
	if err == nil {
		err = e.Patch(c)
	}
	if err != nil {
		return Enumeration{}, err
	}
	return e, nil
}
