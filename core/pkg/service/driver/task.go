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

// Task is the interface that all executable tasks must implement.
type Task interface {
	// Exec handles commands (start, stop, etc.)
	Exec(context.Context, task.Command) error
	// Stop gracefully shuts down the task.
	Stop() error
}

// ErrUnsupportedCommand is returned when a command is not supported by a task.
var ErrUnsupportedCommand = errors.New("unsupported command")
