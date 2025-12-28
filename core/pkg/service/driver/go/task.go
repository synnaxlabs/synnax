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
	"encoding/json"

	"github.com/synnaxlabs/synnax/pkg/service/task"
)

// Task is the interface that all executable tasks must implement.
type Task interface {
	// Exec handles commands (start, stop, etc.)
	Exec(ctx context.Context, cmd Command) error
	// Stop gracefully shuts down the task.
	Stop(ctx context.Context, willReconfigure bool) error
	// Key returns the task key.
	Key() task.Key
}

// Command represents a command to be executed by a task.
type Command struct {
	// Task is the key of the task to execute the command on.
	Task task.Key `json:"task"`
	// Type is the type of command (e.g. "start", "stop").
	Type string `json:"type"`
	// Key is the command key for acknowledgment.
	Key string `json:"key"`
	// Args contains command-specific arguments.
	Args json.RawMessage `json:"args"`
}
