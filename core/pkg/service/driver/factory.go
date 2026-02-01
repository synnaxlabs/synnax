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
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/errors"
)

// Factory is an interface for creating tasks based on their type.
type Factory interface {
	// ConfigureTask creates a task instance if this factory handles the task type.
	// ConfigureTask should return ErrNotHandled if it does not handle the task type.
	ConfigureTask(Context, task.Task) (Task, error)
	// Name returns the factory name for logging.
	Name() string
}

// ErrTaskNotHandled is returned when a task is not handled by a factory.
var ErrTaskNotHandled = errors.New("task not handled by factory")
