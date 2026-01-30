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
)

// CompositeFactory combines multiple factories, delegating to each in order.
type CompositeFactory []Factory

var _ Factory = CompositeFactory{}

// ConfigureTask delegates to each factory in order until one handles the task type.
func (c CompositeFactory) ConfigureTask(ctx Context, t task.Task) (Task, bool, error) {
	for _, f := range c {
		tsk, handled, err := f.ConfigureTask(ctx, t)
		if handled {
			return tsk, true, err
		}
	}
	return nil, false, nil
}

// ConfigureInitialTasks collects initial tasks from all factories.
func (c CompositeFactory) ConfigureInitialTasks(
	ctx Context,
	rackKey rack.Key,
) ([]task.Task, error) {
	var allTasks []task.Task
	for _, f := range c {
		tasks, err := f.ConfigureInitialTasks(ctx, rackKey)
		if err != nil {
			return nil, err
		}
		allTasks = append(allTasks, tasks...)
	}
	return allTasks, nil
}

// Name returns "Composite" as the factory name.
func (c CompositeFactory) Name() string { return "Composite" }
