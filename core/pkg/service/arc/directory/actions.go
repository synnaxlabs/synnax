// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package directory

import (
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/x/patch"
)

// Handle returns an error wrapping validate.ErrValidation, as only the Core creates a
// directory.
func (CreatePayload) Handle(Directory) (Directory, error) {
	return Directory{}, actions.CoreOnlyError("a directory create")
}

// Handle applies the set to the directory's name or parent.
func (p SetPayload) Handle(d Directory) (Directory, error) {
	return edit(d, patch.Set(p).Apply)
}

// Handle applies the remove to the directory's name or parent.
func (p RemovePayload) Handle(d Directory) (Directory, error) {
	return edit(d, patch.Remove(p).Apply)
}

// Handle applies the insert to the directory's name or parent.
func (p InsertPayload) Handle(d Directory) (Directory, error) {
	return edit(d, patch.Insert(p).Apply)
}

// Handle applies the move to the directory's name or parent.
func (p MovePayload) Handle(d Directory) (Directory, error) {
	return edit(d, patch.Move(p).Apply)
}

// document is the part of a directory that path actions edit.
type document struct {
	Name   string `json:"name"`
	Parent *Key   `json:"parent,omitzero"`
}

// edit applies a path action to a copy of the directory's document. A document that
// gains a member it does not declare, such as a path to the directory's key, is
// rejected.
func edit(d Directory, apply func(map[string]any) error) (Directory, error) {
	doc, err := patch.Edit(document{Name: d.Name, Parent: d.Parent}, apply)
	if err != nil {
		return Directory{}, err
	}
	d.Name, d.Parent = doc.Name, doc.Parent
	return d, nil
}
