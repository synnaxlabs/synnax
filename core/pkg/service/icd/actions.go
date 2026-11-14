// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package icd

import (
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/patch"
	"github.com/synnaxlabs/x/validate"
)

// Handle returns an error wrapping validate.ErrValidation, as only the Core creates an
// ICD.
func (CreatePayload) Handle(ICD) (ICD, error) {
	return ICD{}, errors.Wrap(
		validate.ErrValidation, "only the Core dispatches an ICD create",
	)
}

// Handle applies the set to the ICD's name or messages.
func (p SetPayload) Handle(ic ICD) (ICD, error) {
	return edit(ic, patch.Set(p).Apply)
}

// Handle applies the remove to the ICD's name or messages.
func (p RemovePayload) Handle(ic ICD) (ICD, error) {
	return edit(ic, patch.Remove(p).Apply)
}

// Handle applies the insert to the ICD's name or messages.
func (p InsertPayload) Handle(ic ICD) (ICD, error) {
	return edit(ic, patch.Insert(p).Apply)
}

// Handle applies the move to the ICD's name or messages.
func (p MovePayload) Handle(ic ICD) (ICD, error) {
	return edit(ic, patch.Move(p).Apply)
}

// document is the part of an ICD that path actions edit.
type document struct {
	Name     string    `json:"name"`
	Messages []Message `json:"messages"`
}

// edit applies a path action to a copy of the ICD's document. A document that gains a
// member it does not declare, such as a path to the ICD's key, is rejected.
func edit(ic ICD, apply func(map[string]any) error) (ICD, error) {
	d, err := patch.Edit(document{Name: ic.Name, Messages: ic.Messages}, apply)
	if err != nil {
		return ICD{}, err
	}
	ic.Name, ic.Messages = d.Name, d.Messages
	return ic, nil
}
