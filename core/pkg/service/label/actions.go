// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package label

import (
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/x/patch"
)

// Handle returns an error wrapping validate.ErrValidation, as only the Core creates a
// label.
func (CreatePayload) Handle(Label) (Label, error) {
	return Label{}, actions.CoreOnlyError("a label create")
}

// Handle applies the set to the label.
func (p SetPayload) Handle(l Label) (Label, error) {
	return apply(l, patch.Set(p))
}

// Handle applies the remove to the label.
func (p RemovePayload) Handle(l Label) (Label, error) {
	return apply(l, patch.Remove(p))
}

// Handle applies the insert to the label.
func (p InsertPayload) Handle(l Label) (Label, error) {
	return apply(l, patch.Insert(p))
}

// Handle applies the move to the label.
func (p MovePayload) Handle(l Label) (Label, error) {
	return apply(l, patch.Move(p))
}

func apply[A patch.Action](l Label, a A) (Label, error) {
	c, err := patch.Begin(a)
	if err == nil {
		err = l.Patch(c)
	}
	if err != nil {
		return Label{}, err
	}
	return l, nil
}
