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
	"slices"
	"strings"

	"github.com/synnaxlabs/synnax/pkg/service/access"
	"github.com/synnaxlabs/synnax/pkg/service/access/rbac/role"
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/patch"
	"github.com/synnaxlabs/x/validate"
)

// Handle returns an error wrapping validate.ErrValidation, as only the Core creates a
// policy.
func (CreatePayload) Handle(Policy) (Policy, error) {
	return Policy{}, actions.CoreOnlyError("a policy create")
}

// Handle applies the set to the policy's whole name, objects, actions, or role.
func (p SetPayload) Handle(pol Policy) (Policy, error) {
	return edit(pol, patch.Set(p))
}

// Handle returns a validation error, as a policy takes only a set of a whole member.
func (p RemovePayload) Handle(pol Policy) (Policy, error) {
	return edit(pol, patch.Remove(p))
}

// Handle returns a validation error, as a policy takes only a set of a whole member.
func (p InsertPayload) Handle(pol Policy) (Policy, error) {
	return edit(pol, patch.Insert(p))
}

// Handle returns a validation error, as a policy takes only a set of a whole member.
func (p MovePayload) Handle(pol Policy) (Policy, error) {
	return edit(pol, patch.Move(p))
}

// document is the part of a policy that path actions edit. Object keys repeat across
// types and actions are strings, so no list entry has a unique key, and a path action
// may only set one member whole.
type document struct {
	Name    string          `json:"name"`
	Objects []resource.ID   `json:"objects"`
	Actions []access.Action `json:"actions"`
	Role    role.Key        `json:"role"`
}

// edit applies a to a copy of the policy's document. It returns a validation error when
// a is not a set of one whole member, or when the document gains a member it does not
// declare, such as a path to the policy's key.
func edit(p Policy, a patch.Action) (Policy, error) {
	if path, whole := target(a); !whole {
		return Policy{}, errors.Wrapf(
			validate.ErrValidation,
			"[%s]: a policy takes only a set of its whole name, objects, actions, or role",
			strings.Join(path, ", "),
		)
	}
	d, err := patch.Edit(
		document{Name: p.Name, Objects: p.Objects, Actions: p.Actions, Role: p.Role},
		a.Apply,
	)
	if err != nil {
		return Policy{}, err
	}
	p.Name, p.Objects, p.Actions, p.Role = d.Name, d.Objects, d.Actions, d.Role
	return p, nil
}

// target returns the path that a writes, and whether a sets one whole member.
func target(a patch.Action) ([]string, bool) {
	switch a := a.(type) {
	case patch.Set:
		return a.Path, len(a.Path) == 1
	case patch.Remove:
		return slices.Concat(a.Path, []string{a.Key}), false
	case patch.Insert:
		return a.Path, false
	}
	return a.(patch.Move).Path, false
}
