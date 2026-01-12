// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package godriver

import (
	"context"

	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/task"
)

// Context provides shared services to tasks, similar to task::Context in C++.
type Context struct {
	context.Context
	status *status.Service
}

func NewContext(ctx context.Context, statusSvc *status.Service) Context {
	return Context{Context: ctx, status: statusSvc}
}

func (c Context) SetStatus(stat task.Status) error {
	return status.NewWriter[task.StatusDetails](c.status, nil).Set(c.Context, &stat)
}
