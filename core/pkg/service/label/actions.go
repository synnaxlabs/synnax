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
	"github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/patch"
	"github.com/synnaxlabs/x/validate"
)

// Handle returns an error wrapping validate.ErrValidation, as only the Core creates a
// label.
func (CreatePayload) Handle(Label) (Label, error) {
	return Label{}, errors.Wrap(
		validate.ErrValidation, "only the Core dispatches a label create",
	)
}

// Handle applies the set to the label's name or color.
func (p SetPayload) Handle(l Label) (Label, error) {
	return edit(l, patch.Set(p).Apply)
}

// Handle applies the remove to the label's name or color.
func (p RemovePayload) Handle(l Label) (Label, error) {
	return edit(l, patch.Remove(p).Apply)
}

// Handle applies the insert to the label's name or color.
func (p InsertPayload) Handle(l Label) (Label, error) {
	return edit(l, patch.Insert(p).Apply)
}

// Handle applies the move to the label's name or color.
func (p MovePayload) Handle(l Label) (Label, error) {
	return edit(l, patch.Move(p).Apply)
}

// document is the part of a label that path actions edit.
type document struct {
	Name  string      `json:"name"`
	Color color.Color `json:"color"`
}

// edit applies a path action to a copy of the label's document. A document that gains a
// member it does not declare, such as a path to the label's key, is rejected.
func edit(l Label, apply func(map[string]any) error) (Label, error) {
	d, err := patch.Edit(document{Name: l.Name, Color: l.Color}, apply)
	if err != nil {
		return Label{}, err
	}
	l.Name, l.Color = d.Name, d.Color
	return l, nil
}
