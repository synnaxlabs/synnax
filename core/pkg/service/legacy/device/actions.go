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
	"encoding/json/v2"
	"maps"
	"slices"

	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/patch"
)

// Handle returns an error wrapping validate.ErrValidation, as only the Core creates a
// device.
func (CreatePayload) Handle(Device) (Device, error) {
	return Device{}, actions.CoreOnlyError("a device create")
}

// Handle applies the set to the device's document.
func (p SetPayload) Handle(d Device) (Device, error) {
	return edit(d, p.Path, patch.Set(p))
}

// Handle applies the remove to the device's document.
func (p RemovePayload) Handle(d Device) (Device, error) {
	return edit(d, append(slices.Clip(p.Path), p.Key), patch.Remove(p))
}

// Handle applies the insert to the device's document.
func (p InsertPayload) Handle(d Device) (Device, error) {
	return edit(d, p.Path, patch.Insert(p))
}

// Handle applies the move to the device's document.
func (p MovePayload) Handle(d Device) (Device, error) {
	return edit(d, p.Path, patch.Move(p))
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

const propertiesField = "properties"

// edit applies a, which writes path, to a copy of the device's document. A document
// that gains a member it does not declare, such as a path to the device's rack, is
// rejected. An action inside the properties, which hold any JSON, skips the JSON round
// trip of the whole document.
func edit(d Device, path []string, a patch.Action) (Device, error) {
	if len(path) > 1 && path[0] == propertiesField {
		return editProperties(d, path, a)
	}
	doc, err := patch.Edit(documentOf(d), a.Apply)
	if err != nil {
		return Device{}, err
	}
	d.Name, d.Model, d.Location = doc.Name, doc.Model, doc.Location
	d.Configured, d.Properties, d.Group = doc.Configured, doc.Properties, doc.Group
	return d, nil
}

func documentOf(d Device) document {
	return document{
		Name:       d.Name,
		Model:      d.Model,
		Location:   d.Location,
		Configured: d.Configured,
		Properties: d.Properties,
		Group:      d.Group,
	}
}

// editProperties applies a to a copy of the objects and lists on path, so the rest of
// the properties stay shared with d.
func editProperties(d Device, path []string, a patch.Action) (Device, error) {
	// Guards read the other members, so only the properties skip the encode.
	members := documentOf(d)
	members.Properties = nil
	b, err := json.Marshal(members)
	if err != nil {
		return Device{}, err
	}
	var doc map[string]any
	if err = json.Unmarshal(b, &doc); err != nil {
		return Device{}, err
	}
	doc[propertiesField] = copyPath(map[string]any(d.Properties), path[1:])
	if err = a.Apply(doc); err != nil {
		return Device{}, err
	}
	d.Properties = doc[propertiesField].(map[string]any)
	return d, nil
}

// copyPath returns node with each object and list on path copied, including the one at
// its end.
func copyPath(node any, path []string) any {
	switch n := node.(type) {
	case map[string]any:
		c := make(map[string]any, len(n)+1)
		maps.Copy(c, n)
		if len(path) == 0 {
			return c
		}
		if child, ok := c[path[0]]; ok {
			c[path[0]] = copyPath(child, path[1:])
		}
		return c
	case []any:
		c := slices.Clone(n)
		if len(path) == 0 {
			return c
		}
		// A path segment into a list names the key of an entry.
		for i, entry := range c {
			if m, ok := entry.(map[string]any); ok && m["key"] == path[0] {
				c[i] = copyPath(m, path[1:])
			}
		}
		return c
	default:
		return node
	}
}
