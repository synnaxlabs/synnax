// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package symbol

import (
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/x/patch"
)

// Handle returns an error wrapping validate.ErrValidation, as only the Core creates a
// symbol.
func (CreatePayload) Handle(Symbol) (Symbol, error) {
	return Symbol{}, actions.CoreOnlyError("a symbol create")
}

// Handle applies the set to the symbol's name, data, or group.
func (p SetPayload) Handle(s Symbol) (Symbol, error) {
	return edit(s, patch.Set(p).Apply)
}

// Handle applies the remove to the symbol's name, data, or group.
func (p RemovePayload) Handle(s Symbol) (Symbol, error) {
	return edit(s, patch.Remove(p).Apply)
}

// Handle applies the insert to the symbol's name, data, or group.
func (p InsertPayload) Handle(s Symbol) (Symbol, error) {
	return edit(s, patch.Insert(p).Apply)
}

// Handle applies the move to the symbol's name, data, or group.
func (p MovePayload) Handle(s Symbol) (Symbol, error) {
	return edit(s, patch.Move(p).Apply)
}

// document is the part of a symbol that path actions edit.
type document struct {
	Name  string     `json:"name"`
	Data  Spec       `json:"data"`
	Group *group.Key `json:"group,omitzero"`
}

// edit applies a path action to a copy of the symbol's document. A document that gains
// a member it does not declare, such as a path to the symbol's key, is rejected.
func edit(s Symbol, apply func(map[string]any) error) (Symbol, error) {
	d, err := patch.Edit(document{Name: s.Name, Data: s.Data, Group: s.Group}, apply)
	if err != nil {
		return Symbol{}, err
	}
	s.Name, s.Data, s.Group = d.Name, d.Data, d.Group
	return s, nil
}
