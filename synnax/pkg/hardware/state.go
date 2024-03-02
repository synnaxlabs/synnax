// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package hardware

import (
	"github.com/synnaxlabs/synnax/pkg/hardware/rack"
	"github.com/synnaxlabs/synnax/pkg/hardware/task"
)

type RackState struct {
	Key       rack.Key
	Heartbeat uint32
}

type TaskStatus string

const (
	TaskStatusPending TaskStatus = "stopped"
	TaskStatusRunning TaskStatus = "running"
	TaskStatusStopped TaskStatus = "failed"
)

type TaskState struct {
	Key    task.Key
	Status TaskStatus

	Error error
}

type State struct {
	Racks map[rack.Key]RackState
}
