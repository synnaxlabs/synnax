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
	// Stop gracefully shuts down the task. When sendStatus is false the task
	// emits no terminal status: a successor instance owns status reporting.
	Stop(sendStatus bool) error
}

var (
	// ErrUnsupportedCommand is returned when a command is not supported by a task.
	ErrUnsupportedCommand = errors.New("unsupported command")
	// ErrTemporary marks an error that a task recovers from by trying again, such as a
	// lost connection. A task reports it as a warning and keeps running. Mark an error
	// with errors.Mark, so that the status shows the message of the error alone.
	ErrTemporary = errors.New("temporary error")
	// ErrDegraded marks a condition that loses data while the task keeps up its work,
	// such as a full queue. A task reports it as a warning and continues at once. Mark
	// an error with errors.Mark, so that the status shows the message of the error
	// alone.
	ErrDegraded = errors.New("degraded")
)
