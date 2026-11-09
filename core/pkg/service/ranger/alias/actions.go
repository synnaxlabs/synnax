// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package alias

import (
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/x/patch"
)

// Handle returns an error wrapping validate.ErrValidation, as only the Core creates an
// alias.
func (CreatePayload) Handle(Alias) (Alias, error) {
	return Alias{}, actions.CoreOnlyError("an alias create")
}

// Handle applies the set to the alias text.
func (p SetPayload) Handle(
	a Alias,
) (Alias, error) {
	return edit(a, patch.Set(p).Apply)
}

// Handle applies the remove to the alias text.
func (p RemovePayload) Handle(a Alias) (Alias, error) {
	return edit(a, patch.Remove(p).Apply)
}

// Handle applies the insert to the alias text.
func (p InsertPayload) Handle(a Alias) (Alias, error) {
	return edit(a, patch.Insert(p).Apply)
}

// Handle applies the move to the alias text.
func (p MovePayload) Handle(
	a Alias,
) (Alias, error) {
	return edit(a, patch.Move(p).Apply)
}

// document is the part of an alias that path actions edit.
type document struct {
	Alias string `json:"alias"`
}

// edit applies a path action to a copy of the alias's document. A document that gains a
// member it does not declare, such as a path to the alias's channel, is rejected.
func edit(a Alias, apply func(map[string]any) error) (Alias, error) {
	d, err := patch.Edit(document{Alias: a.Alias}, apply)
	if err != nil {
		return Alias{}, err
	}
	a.Alias = d.Alias
	return a, nil
}
