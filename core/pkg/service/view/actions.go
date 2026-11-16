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
	"github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/patch"
)

// Handle returns an error wrapping validate.ErrValidation, as only the Core creates a
// view.
func (CreatePayload) Handle(View) (View, error) {
	return View{}, actions.CoreOnlyError("a view create")
}

// Handle applies the set to the view's document.
func (p SetPayload) Handle(v View) (View, error) {
	return edit(v, patch.Set(p).Apply)
}

// Handle applies the remove to the view's document.
func (p RemovePayload) Handle(v View) (View, error) {
	return edit(v, patch.Remove(p).Apply)
}

// Handle applies the insert to the view's document.
func (p InsertPayload) Handle(v View) (View, error) {
	return edit(v, patch.Insert(p).Apply)
}

// Handle applies the move to the view's document.
func (p MovePayload) Handle(v View) (View, error) {
	return edit(v, patch.Move(p).Apply)
}

// document is the part of a view that path actions edit. W is the where of the view's
// variant.
type document[W any] struct {
	Name  string       `json:"name"`
	Icon  *string      `json:"icon,omitzero"`
	Color *color.Color `json:"color,omitzero"`
	Where *W           `json:"where,omitzero"`
}

func newDocument[W any](b Base, where *W) document[W] {
	return document[W]{Name: b.Name, Icon: b.Icon, Color: b.Color, Where: where}
}

func (d document[W]) base(key Key) Base {
	return Base{Key: key, Name: d.Name, Icon: d.Icon, Color: d.Color}
}

// rangeDocument is the part of a range view that path actions edit.
type rangeDocument struct {
	document[ranger.Where]
	Sort   RangeSort   `json:"sort"`
	Layout RangeLayout `json:"layout"`
}

// edit applies a path action to a copy of the view's document. A document that gains a
// member it does not declare, such as a path to the view's type, is rejected.
func edit(v View, apply func(map[string]any) error) (View, error) {
	switch variant := v.Variant.(type) {
	case RangeView:
		d, err := patch.Edit(
			rangeDocument{
				document: newDocument(variant.Base, variant.Where),
				Sort:     variant.Sort,
				Layout:   variant.Layout,
			},
			apply,
		)
		if err != nil {
			return View{}, err
		}
		variant.Base, variant.Where = d.base(variant.Key), d.Where
		variant.Sort, variant.Layout = d.Sort, d.Layout
		return View{Variant: variant}, nil
	case StatusView:
		d, err := patch.Edit(newDocument(variant.Base, variant.Where), apply)
		if err != nil {
			return View{}, err
		}
		variant.Base, variant.Where = d.base(variant.Key), d.Where
		return View{Variant: variant}, nil
	}
	return View{}, errors.Newf(
		"view %s has the unknown variant %T",
		v.Base().Key,
		v.Variant,
	)
}
