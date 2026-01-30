// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package driver

import (
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/errors"
)

// Factory is an interface for creating tasks based on their type.
type Factory interface {
	// ConfigureTask creates a task instance if this factory handles the task type.
	// Returns the task instance if handled successfully, ErrNotHandled if the factory
	// does not handle the task type, or another error if the factory handles the task
	// type but configuration failed.
	ConfigureTask(Context, task.Task) (Task, error)
	// ConfigureInitialTasks creates tasks that should exist on startup for the given
	// rack. Called once during driver initialization. The factory should query its own
	// device service to find relevant devices and create appropriate tasks. Returns
	// task definitions to be persisted and configured, or an error.
	ConfigureInitialTasks(Context, rack.Key) ([]task.Task, error)
}

var ErrNotHandled = errors.New("task not handled by factory")
