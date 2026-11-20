// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package rack

import (
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/x/patch"
)

// Handle returns an error wrapping validate.ErrValidation, as only the Core creates a
// rack.
func (CreatePayload) Handle(Rack) (Rack, error) {
	return Rack{}, actions.CoreOnlyError("a rack create")
}

// Handle applies the set to the rack's document.
func (p SetPayload) Handle(r Rack) (Rack, error) {
	return edit(r, patch.Set(p).Apply)
}

// Handle applies the remove to the rack's document.
func (p RemovePayload) Handle(r Rack) (Rack, error) {
	return edit(r, patch.Remove(p).Apply)
}

// Handle applies the insert to the rack's document.
func (p InsertPayload) Handle(r Rack) (Rack, error) {
	return edit(r, patch.Insert(p).Apply)
}

// Handle applies the move to the rack's document.
func (p MovePayload) Handle(r Rack) (Rack, error) {
	return edit(r, patch.Move(p).Apply)
}

// document is the part of a rack that path actions edit: its name, integrations, Driver
// version, and group.
type document struct {
	Name         string     `json:"name"`
	Integrations []string   `json:"integrations"`
	Version      *string    `json:"version,omitzero"`
	Group        *group.Key `json:"group,omitzero"`
}

// edit applies a path action to a copy of the rack's document. A document that gains a
// member it does not declare, such as a path to the rack's role, is rejected.
func edit(r Rack, apply func(map[string]any) error) (Rack, error) {
	d, err := patch.Edit(
		document{
			Name:         r.Name,
			Integrations: r.Integrations,
			Version:      r.Version,
			Group:        r.Group,
		},
		apply,
	)
	if err != nil {
		return Rack{}, err
	}
	r.Name, r.Integrations = d.Name, d.Integrations
	r.Version, r.Group = d.Version, d.Group
	return r, nil
}
