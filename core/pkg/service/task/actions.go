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
	"github.com/synnaxlabs/synnax/pkg/service/actions"
	"github.com/synnaxlabs/x/patch"
)

// Handle returns an error wrapping validate.ErrValidation, as only the Core creates a
// task.
func (CreatePayload) Handle(Task) (Task, error) {
	return Task{}, actions.CoreOnlyError("a task create")
}

// Handle applies the set to the task's name or rack.
func (p SetPayload) Handle(t Task) (Task, error) { return apply(t, patch.Set(p)) }

// Handle applies the remove to the task's name or rack.
func (p RemovePayload) Handle(t Task) (Task, error) { return apply(t, patch.Remove(p)) }

// Handle applies the insert to the task's name or rack.
func (p InsertPayload) Handle(t Task) (Task, error) { return apply(t, patch.Insert(p)) }

// Handle applies the move to the task's name or rack.
func (p MovePayload) Handle(t Task) (Task, error) { return apply(t, patch.Move(p)) }

// apply applies a path action to the task's name or rack. The config store of the
// task's type applies the actions on its config.
func apply[A patch.Action](t Task, a A) (Task, error) {
	c, err := patch.Begin(a)
	if err == nil {
		err = t.Patch(c)
	}
	if err != nil {
		return Task{}, err
	}
	return t, nil
}

// cursor returns the cursor of a path action, and false for a create.
func (a Action) cursor() (patch.Cursor, bool, error) {
	var (
		c   patch.Cursor
		err error
	)
	switch {
	case a.Set != nil:
		c, err = patch.Begin(patch.Set(*a.Set))
	case a.Remove != nil:
		c, err = patch.Begin(patch.Remove(*a.Remove))
	case a.Insert != nil:
		c, err = patch.Begin(patch.Insert(*a.Insert))
	case a.Move != nil:
		c, err = patch.Begin(patch.Move(*a.Move))
	default:
		return patch.Cursor{}, false, nil
	}
	return c, true, err
}
