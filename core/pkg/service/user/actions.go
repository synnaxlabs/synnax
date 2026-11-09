// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package user

import (
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/x/patch"
)

// Handle returns an error wrapping validate.ErrValidation, as only the Core creates a
// user.
func (CreatePayload) Handle(User) (User, error) {
	return User{}, actions.CoreOnlyError("a user create")
}

// Handle applies the set to the user's name or role.
func (p SetPayload) Handle(u User) (User, error) {
	return edit(u, patch.Set(p).Apply)
}

// Handle applies the remove to the user's name or role.
func (p RemovePayload) Handle(u User) (User, error) {
	return edit(u, patch.Remove(p).Apply)
}

// Handle applies the insert to the user's name or role.
func (p InsertPayload) Handle(u User) (User, error) {
	return edit(u, patch.Insert(p).Apply)
}

// Handle applies the move to the user's name or role.
func (p MovePayload) Handle(u User) (User, error) {
	return edit(u, patch.Move(p).Apply)
}

// Handle returns u unchanged, as the username lives on the user's password.
// Writer.Apply renames the password.
func (SetUsernamePayload) Handle(u User) (User, error) { return u, nil }

// document is the part of a user that path actions edit.
type document struct {
	Name string   `json:"name"`
	Role role.Key `json:"role"`
}

// edit applies a path action to a copy of the user's document. A document that gains a
// member it does not declare, such as a path to root_user, is rejected.
func edit(u User, apply func(map[string]any) error) (User, error) {
	d, err := patch.Edit(document{Name: u.Name, Role: u.Role}, apply)
	if err != nil {
		return User{}, err
	}
	u.Name, u.Role = d.Name, d.Role
	return u, nil
}

func newSetAction(path string, value any) Action {
	return NewSetAction(SetPayload{Path: []string{path}, Value: value})
}
