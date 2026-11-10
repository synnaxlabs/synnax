// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package view

import (
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/patch"
)

// Handle returns an error wrapping validate.ErrValidation, as only the Core creates a
// view.
func (CreatePayload) Handle(View) (View, error) {
	return View{}, actions.CoreOnlyError("a view create")
}

// Handle applies the set to the view's name or where.
func (p SetPayload) Handle(v View) (View, error) {
	return edit(v, patch.Set(p).Apply)
}

// Handle applies the remove to the view's name or where.
func (p RemovePayload) Handle(v View) (View, error) {
	return edit(v, patch.Remove(p).Apply)
}

// Handle applies the insert to the view's name or where.
func (p InsertPayload) Handle(v View) (View, error) {
	return edit(v, patch.Insert(p).Apply)
}

// Handle applies the move to the view's name or where.
func (p MovePayload) Handle(v View) (View, error) {
	return edit(v, patch.Move(p).Apply)
}

// document is the part of a view that path actions edit. W is the where of the view's
// variant.
type document[W any] struct {
	Name  string `json:"name"`
	Where *W     `json:"where,omitzero"`
}

// edit applies a path action to a copy of the view's document. A document that gains a
// member it does not declare, such as a path to the view's type, is rejected.
func edit(v View, apply func(map[string]any) error) (View, error) {
	switch variant := v.Variant.(type) {
	case RangeView:
		d, err := patch.Edit(
			document[ranger.Where]{Name: variant.Name, Where: variant.Where},
			apply,
		)
		if err != nil {
			return View{}, err
		}
		variant.Name, variant.Where = d.Name, d.Where
		return View{Variant: variant}, nil
	case StatusView:
		d, err := patch.Edit(
			document[status.Where]{Name: variant.Name, Where: variant.Where},
			apply,
		)
		if err != nil {
			return View{}, err
		}
		variant.Name, variant.Where = d.Name, d.Where
		return View{Variant: variant}, nil
	}
	return View{}, errors.Newf(
		"view %s has the unknown variant %T",
		v.Base().Key,
		v.Variant,
	)
}
