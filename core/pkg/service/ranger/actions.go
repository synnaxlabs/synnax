// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ranger

import (
	"github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/patch"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// Handle returns an error wrapping validate.ErrValidation, as only the Core creates a
// range.
func (CreatePayload) Handle(Range) (Range, error) {
	return Range{}, errors.Wrap(
		validate.ErrValidation, "only the Core dispatches a range create",
	)
}

// Handle applies the set to the range's name, time range, or color.
func (p SetPayload) Handle(r Range) (Range, error) {
	return edit(r, patch.Set(p).Apply)
}

// Handle applies the remove to the range's name, time range, or color.
func (p RemovePayload) Handle(r Range) (Range, error) {
	return edit(r, patch.Remove(p).Apply)
}

// Handle applies the insert to the range's name, time range, or color.
func (p InsertPayload) Handle(r Range) (Range, error) {
	return edit(r, patch.Insert(p).Apply)
}

// Handle applies the move to the range's name, time range, or color.
func (p MovePayload) Handle(r Range) (Range, error) {
	return edit(r, patch.Move(p).Apply)
}

// document is the part of a range that path actions edit.
type document struct {
	Name      string          `json:"name"`
	TimeRange telem.TimeRange `json:"time_range"`
	Color     *color.Color    `json:"color,omitzero"`
}

// edit applies a path action to a copy of the range's document. A document that gains a
// member it does not declare, such as a path to the range's labels, is rejected.
func edit(r Range, apply func(map[string]any) error) (Range, error) {
	d, err := patch.Edit(
		document{Name: r.Name, TimeRange: r.TimeRange, Color: r.Color},
		apply,
	)
	if err != nil {
		return Range{}, err
	}
	r.Name, r.TimeRange, r.Color = d.Name, d.TimeRange, d.Color
	return r, nil
}
