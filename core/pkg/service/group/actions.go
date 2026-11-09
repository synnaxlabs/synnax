// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package group

import (
	"github.com/synnaxlabs/x/patch"
)

// Handle applies the set to the group's name.
func (p SetPayload) Handle(g Group) (Group, error) {
	return edit(g, patch.Set(p).Apply)
}

// Handle applies the remove to the group's name.
func (p RemovePayload) Handle(g Group) (Group, error) {
	return edit(g, patch.Remove(p).Apply)
}

// Handle applies the insert to the group's name.
func (p InsertPayload) Handle(g Group) (Group, error) {
	return edit(g, patch.Insert(p).Apply)
}

// Handle applies the move to the group's name.
func (p MovePayload) Handle(g Group) (Group, error) {
	return edit(g, patch.Move(p).Apply)
}

// Handle sets the group's parent. Writer.Apply checks the move and moves the records in
// the group.
func (p SetParentPayload) Handle(g Group) (Group, error) {
	g.Parent = p.Parent
	return g, nil
}

// document is the part of a group that path actions edit.
type document struct {
	Name string `json:"name"`
}

// edit applies a path action to a copy of the group's document. A document that gains a
// member it does not declare, such as a path to the group's parent, is rejected.
func edit(g Group, apply func(map[string]any) error) (Group, error) {
	d, err := patch.Edit(document{Name: g.Name}, apply)
	if err != nil {
		return Group{}, err
	}
	g.Name = d.Name
	return g, nil
}
