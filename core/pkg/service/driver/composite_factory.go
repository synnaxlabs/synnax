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

// CompositeFactory combines multiple factories, delegating to each in order.
type CompositeFactory []Factory

var _ Factory = CompositeFactory{}

// ConfigureTask delegates to each factory in order until one handles the task type.
func (c CompositeFactory) ConfigureTask(ctx Context, t task.Task) (Task, error) {
	for _, f := range c {
		if tsk, err := f.ConfigureTask(ctx, t); !errors.Is(err, ErrNotHandled) {
			return tsk, err
		}
	}
	return nil, ErrNotHandled
}

// ConfigureInitialTasks collects initial tasks from all factories.
func (c CompositeFactory) ConfigureInitialTasks(
	ctx Context,
	rack rack.Key,
) ([]task.Task, error) {
	var allTasks []task.Task
	for _, f := range c {
		tasks, err := f.ConfigureInitialTasks(ctx, rack)
		if err != nil {
			return nil, err
		}
		allTasks = append(allTasks, tasks...)
	}
	return allTasks, nil
}
