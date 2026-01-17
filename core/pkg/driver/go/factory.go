// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package godriver

import (
	"github.com/synnaxlabs/synnax/pkg/service/task"
)

// Factory is an interface for creating tasks based on their type.
type Factory interface {
	// ConfigureTask creates a task instance if this factory handles the task type.
	// Returns (task, true, nil) if handled successfully.
	// Returns (nil, false, nil) if this factory does not handle the task type.
	// Returns (nil, true, err) if the factory handles this type but configuration failed.
	ConfigureTask(ctx Context, t task.Task) (Task, bool, error)
	// Name returns the factory name for logging.
	Name() string
}

// MultiFactory chains multiple factories together, trying each one in order
// until one handles the task.
type MultiFactory []Factory

// ConfigureTask tries each factory in order until one handles the task.
func (m MultiFactory) ConfigureTask(ctx Context, t task.Task) (Task, bool, error) {
	for _, f := range m {
		if tsk, ok, err := f.ConfigureTask(ctx, t); ok {
			return tsk, ok, err
		} else if err != nil {
			return nil, true, err
		}
	}
	return nil, false, nil
}

// Name returns "multi" as this is a composite factory.
func (m *MultiFactory) Name() string { return "multi" }
