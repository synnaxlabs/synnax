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

	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/telem"
)

// ContextOption is a functional option for configuring a Context.
type ContextOption func(*Context)

// WithStatusService sets the status service on the context.
func WithStatusService(svc *status.Service) ContextOption {
	return func(c *Context) { c.statusSvc = svc }
}

// WithRegister sets a callback that registers the task with the driver's command
// dispatch map. Factories should call ctx.Register before emitting any status that
// signals the task is ready for commands.
func WithRegister(fn func(Task)) ContextOption {
	return func(c *Context) { c.registerFn = fn }
}

// Context provides shared services to tasks, similar to task::Context in C++.
type Context struct {
	context.Context
	statusSvc  *status.Service
	registerFn func(Task)
}

// NewContext creates a new Context with the given options.
func NewContext(ctx context.Context, opts ...ContextOption) Context {
	c := Context{Context: ctx}
	for _, opt := range opts {
		opt(&c)
	}
	return c
}

// SetStatus updates the status of the task, auto-filling the time if not provided.
func (c Context) SetStatus(stat task.Status) error {
	if c.statusSvc == nil {
		return nil
	}
	if stat.Time == 0 {
		stat.Time = telem.Now()
	}
	return status.NewWriter[task.StatusDetails](c.statusSvc, nil).Set(c.Context, &stat)
}

// Register registers the task with the driver's command dispatch map. Factories should
// call this before emitting any status that signals the task is ready for commands
// (e.g., "configured" or "started"). This ensures that commands sent in response to the
// status update can find the task immediately.
func (c Context) Register(t Task) {
	if c.registerFn != nil {
		c.registerFn(t)
	}
}
