// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package role

import (
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/x/patch"
)

// Handle returns an error wrapping validate.ErrValidation, as only the Core creates a
// role.
func (CreatePayload) Handle(Role) (Role, error) {
	return Role{}, actions.CoreOnlyError("a role create")
}

// Handle applies the set to the role's name or description.
func (p SetPayload) Handle(r Role) (Role, error) {
	return edit(r, patch.Set(p).Apply)
}

// Handle applies the remove to the role's name or description.
func (p RemovePayload) Handle(r Role) (Role, error) {
	return edit(r, patch.Remove(p).Apply)
}

// Handle applies the insert to the role's name or description.
func (p InsertPayload) Handle(r Role) (Role, error) {
	return edit(r, patch.Insert(p).Apply)
}

// Handle applies the move to the role's name or description.
func (p MovePayload) Handle(r Role) (Role, error) {
	return edit(r, patch.Move(p).Apply)
}

// document is the part of a role that path actions edit.
type document struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// edit applies a path action to a copy of the role's document. A document that gains a
// member it does not declare, such as a path to internal, is rejected.
func edit(r Role, apply func(map[string]any) error) (Role, error) {
	d, err := patch.Edit(document{Name: r.Name, Description: r.Description}, apply)
	if err != nil {
		return Role{}, err
	}
	r.Name, r.Description = d.Name, d.Description
	return r, nil
}
