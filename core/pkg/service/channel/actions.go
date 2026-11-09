// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel

import (
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/x/patch"
)

// Handle applies the set to the channel's expression, operations, or group.
func (p SetPayload) Handle(c Channel) (Channel, error) {
	return edit(c, patch.Set(p).Apply)
}

// Handle applies the remove to the channel's expression, operations, or group.
func (p RemovePayload) Handle(c Channel) (Channel, error) {
	return edit(c, patch.Remove(p).Apply)
}

// Handle applies the insert to the channel's expression, operations, or group.
func (p InsertPayload) Handle(c Channel) (Channel, error) {
	return edit(c, patch.Insert(p).Apply)
}

// Handle applies the move to the channel's expression, operations, or group.
func (p MovePayload) Handle(c Channel) (Channel, error) {
	return edit(c, patch.Move(p).Apply)
}

// Handle gives the channel the name of p.
func (p RenamePayload) Handle(c Channel) (Channel, error) {
	c.Name = p.Name
	return c, nil
}

// document is the part of a channel that path actions edit.
type document struct {
	Expression string      `json:"expression"`
	Operations []Operation `json:"operations"`
	Group      *group.Key  `json:"group,omitzero"`
}

// edit applies a path action to a copy of the channel's document. A document that gains
// a member it does not declare, such as a path to the channel's data type, is rejected.
func edit(c Channel, apply func(map[string]any) error) (Channel, error) {
	d, err := patch.Edit(
		document{Expression: c.Expression, Operations: c.Operations, Group: c.Group},
		apply,
	)
	if err != nil {
		return Channel{}, err
	}
	c.Expression, c.Operations, c.Group = d.Expression, d.Operations, d.Group
	return c, nil
}
