// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package policy

import (
	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/patch"
)

// Handle returns an error wrapping validate.ErrValidation, as only the Core creates a
// policy.
func (CreatePayload) Handle(Policy) (Policy, error) {
	return Policy{}, actions.CoreOnlyError("a policy create")
}

// Handle applies the set to the policy's name, objects, or actions.
func (p SetPayload) Handle(pol Policy) (Policy, error) {
	return edit(pol, patch.Set(p).Apply)
}

// Handle applies the remove to the policy's name, objects, or actions.
func (p RemovePayload) Handle(pol Policy) (Policy, error) {
	return edit(pol, patch.Remove(p).Apply)
}

// Handle applies the insert to the policy's name, objects, or actions.
func (p InsertPayload) Handle(pol Policy) (Policy, error) {
	return edit(pol, patch.Insert(p).Apply)
}

// Handle applies the move to the policy's name, objects, or actions.
func (p MovePayload) Handle(pol Policy) (Policy, error) {
	return edit(pol, patch.Move(p).Apply)
}

// document is the part of a policy that path actions edit.
type document struct {
	Name    string          `json:"name"`
	Objects []resource.ID   `json:"objects"`
	Actions []access.Action `json:"actions"`
}

// edit applies a path action to a copy of the policy's document. A document that gains
// a member it does not declare, such as a path to the policy's role, is rejected.
func edit(p Policy, apply func(map[string]any) error) (Policy, error) {
	d, err := patch.Edit(
		document{Name: p.Name, Objects: p.Objects, Actions: p.Actions},
		apply,
	)
	if err != nil {
		return Policy{}, err
	}
	p.Name, p.Objects, p.Actions = d.Name, d.Objects, d.Actions
	return p, nil
}
