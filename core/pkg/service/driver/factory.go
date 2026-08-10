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
	"context"

	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/errors"
)

// NoCommand is the cmdKey given to ConfigureTask when no command drives the
// configure, as at boot. Nothing is waiting on the outcome.
const NoCommand = ""

// Factory is an interface for creating tasks based on their type.
type Factory interface {
	// ConfigureTask creates a task instance if this factory handles the task type.
	// ConfigureTask should return ErrNotHandled if it does not handle the task type.
	// cmdKey is the start command driving the deploy, NoCommand at boot. A factory
	// returning any other error must first write a status carrying cmdKey: that
	// status is what answers the caller waiting on the command. Given NoCommand
	// there is no caller, so failures on tasks that do not auto-start are logged.
	ConfigureTask(ctx context.Context, t task.Task, cmdKey string) (Task, error)
	// Name returns the integration name of this factory. This is used to identify the
	// integrations allowed on the rack.
	Name() string
}

// ErrTaskNotHandled is returned when a task is not handled by a factory.
var ErrTaskNotHandled = errors.New("task not handled by factory")
