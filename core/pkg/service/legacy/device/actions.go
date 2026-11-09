// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package device

import (
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/patch"
)

// Handle applies the set to the device's document.
func (p SetPayload) Handle(d Device) (Device, error) {
	return edit(d, patch.Set(p).Apply)
}

// Handle applies the remove to the device's document.
func (p RemovePayload) Handle(d Device) (Device, error) {
	return edit(d, patch.Remove(p).Apply)
}

// Handle applies the insert to the device's document.
func (p InsertPayload) Handle(d Device) (Device, error) {
	return edit(d, patch.Insert(p).Apply)
}

// Handle applies the move to the device's document.
func (p MovePayload) Handle(d Device) (Device, error) {
	return edit(d, patch.Move(p).Apply)
}

// Handle moves the device to the rack and parent of p, and out of its group.
func (p SetRackPayload) Handle(d Device) (Device, error) {
	d.Rack, d.Parent, d.Group = p.Rack, p.Parent, nil
	return d, nil
}

// document is the part of a device that path actions edit: its name, model, location,
// configured flag, properties, and group.
type document struct {
	Name       string              `json:"name"`
	Model      string              `json:"model"`
	Location   string              `json:"location"`
	Configured bool                `json:"configured"`
	Properties msgpack.EncodedJSON `json:"properties"`
	Group      *group.Key          `json:"group,omitzero"`
}

// edit applies a path action to a copy of the device's document. A document that gains
// a member it does not declare, such as a path to the device's rack, is rejected.
func edit(d Device, apply func(map[string]any) error) (Device, error) {
	doc, err := patch.Edit(
		document{
			Name:       d.Name,
			Model:      d.Model,
			Location:   d.Location,
			Configured: d.Configured,
			Properties: d.Properties,
			Group:      d.Group,
		},
		apply,
	)
	if err != nil {
		return Device{}, err
	}
	d.Name, d.Model, d.Location = doc.Name, doc.Model, doc.Location
	d.Configured, d.Properties, d.Group = doc.Configured, doc.Properties, doc.Group
	return d, nil
}
