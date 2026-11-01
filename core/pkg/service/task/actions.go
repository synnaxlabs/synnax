// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package task

import (
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/patch"
	"github.com/synnaxlabs/x/validate"
)

// Handle returns an error wrapping validate.ErrValidation, as only the Core creates a
// task.
func (CreatePayload) Handle(Task) (Task, error) {
	return Task{}, errors.Wrap(
		validate.ErrValidation, "only the Core dispatches a task create",
	)
}

// Handle applies the set to the task's name, rack, or config.
func (p SetPayload) Handle(t Task) (Task, error) {
	return edit(t, patch.Set(p).Apply)
}

// Handle applies the remove to the task's name, rack, or config.
func (p RemovePayload) Handle(t Task) (Task, error) {
	return edit(t, patch.Remove(p).Apply)
}

// Handle applies the insert to the task's name, rack, or config.
func (p InsertPayload) Handle(t Task) (Task, error) {
	return edit(t, patch.Insert(p).Apply)
}

// Handle applies the move to the task's name, rack, or config.
func (p MovePayload) Handle(t Task) (Task, error) {
	return edit(t, patch.Move(p).Apply)
}

// document is the part of a task that path actions edit.
type document struct {
	Name   string              `json:"name"`
	Rack   rack.Key            `json:"rack"`
	Config msgpack.EncodedJSON `json:"config"`
}

// edit applies a path action to a copy of the task's document. A document that gains a
// member it does not declare, such as a path to the task's type, is rejected.
func edit(t Task, apply func(map[string]any) error) (Task, error) {
	d, err := patch.Edit(document{Name: t.Name, Rack: t.Rack, Config: t.Config}, apply)
	if err != nil {
		return Task{}, err
	}
	t.Name, t.Rack, t.Config = d.Name, d.Rack, d.Config
	return t, nil
}
