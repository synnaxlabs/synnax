// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package secret

import (
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/x/patch"
)

// Handle returns an error wrapping validate.ErrValidation, as only the Core creates a
// secret.
func (CreatePayload) Handle(Secret) (Secret, error) {
	return Secret{}, actions.CoreOnlyError("a secret create")
}

// Handle applies the set to the secret's name.
func (p SetPayload) Handle(s Secret) (Secret, error) {
	return edit(s, patch.Set(p).Apply)
}

// Handle applies the remove to the secret's name.
func (p RemovePayload) Handle(s Secret) (Secret, error) {
	return edit(s, patch.Remove(p).Apply)
}

// Handle applies the insert to the secret's name.
func (p InsertPayload) Handle(s Secret) (Secret, error) {
	return edit(s, patch.Insert(p).Apply)
}

// Handle applies the move to the secret's name.
func (p MovePayload) Handle(s Secret) (Secret, error) {
	return edit(s, patch.Move(p).Apply)
}

// document is the part of a secret that path actions edit. The value and file name
// change only through a create, which replaces the secret.
type document struct {
	Name string `json:"name"`
}

// edit applies a path action to a copy of the secret's document. A document that gains
// a member it does not declare, such as a path to the secret's value, is rejected.
func edit(s Secret, apply func(map[string]any) error) (Secret, error) {
	base := s.Base()
	d, err := patch.Edit(document{Name: base.Name}, apply)
	if err != nil {
		return Secret{}, err
	}
	base.Name = d.Name
	s.SetBase(base)
	return s, nil
}
