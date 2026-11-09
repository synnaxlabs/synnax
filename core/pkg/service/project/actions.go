// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package project

import (
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/x/patch"
)

// Handle applies the set to the project's name or group.
func (p SetPayload) Handle(proj Project) (Project, error) {
	return edit(proj, patch.Set(p).Apply)
}

// Handle applies the remove to the project's name or group.
func (p RemovePayload) Handle(proj Project) (Project, error) {
	return edit(proj, patch.Remove(p).Apply)
}

// Handle applies the insert to the project's name or group.
func (p InsertPayload) Handle(proj Project) (Project, error) {
	return edit(proj, patch.Insert(p).Apply)
}

// Handle applies the move to the project's name or group.
func (p MovePayload) Handle(proj Project) (Project, error) {
	return edit(proj, patch.Move(p).Apply)
}

// document is the part of a project that path actions edit.
type document struct {
	Name  string     `json:"name"`
	Group *group.Key `json:"group,omitzero"`
}

// edit applies a path action to a copy of the project's document. A document that gains
// a member it does not declare, such as a path to the project's key, is rejected.
func edit(p Project, apply func(map[string]any) error) (Project, error) {
	d, err := patch.Edit(document{Name: p.Name, Group: p.Group}, apply)
	if err != nil {
		return Project{}, err
	}
	p.Name, p.Group = d.Name, d.Group
	return p, nil
}
