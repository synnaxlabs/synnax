// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0

import (
	"strconv"

	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	status "github.com/synnaxlabs/synnax/pkg/service/status/migrations/v0"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/gorp"
)

// Key is a composite identifier for a task. The high 32 bits contain the rack key, and
// the low 32 bits contain the local task key within that rack.
type Key uint64

func (k Key) String() string { return strconv.FormatUint(uint64(k), 10) }

// Status is task-specific status information including execution state and
// task-specific data.
type Status = status.Status[StatusDetails]

// StatusDetails contains task-specific status details including execution state.
type StatusDetails struct {
	// Task is the key of the task this status pertains to.
	Task Key `json:"task" msgpack:"task"`
	// Running is true if the task is currently executing.
	Running bool `json:"running" msgpack:"running"`
	// Cmd is the last command executed on this task.
	Cmd string `json:"cmd" msgpack:"cmd"`
	// Data contains task-specific status data.
	Data msgpack.EncodedJSON `json:"data,omitempty" msgpack:"data,omitempty"`
}

// Task is an executable unit of work in the Driver system. Tasks represent specific
// hardware operations such as reading sensor data, writing control signals, or scanning
// for devices.
type Task struct {
	// Key is the composite identifier for this task.
	Key Key `json:"key" msgpack:"key"`
	// Name is a human-readable name for the task.
	Name string `json:"name" msgpack:"name"`
	// Type is the task type (e.g., 'modbus_read', 'labjack_write', 'opc_scan'). Determines
	// which hardware integration handles the task.
	Type string `json:"type" msgpack:"type"`
	// Config is task-specific configuration stored as JSON. Structure varies by task type.
	Config msgpack.EncodedJSON `json:"config" msgpack:"config"`
	// Internal is true if this is an internal system task.
	Internal bool `json:"internal" msgpack:"internal"`
	// Snapshot indicates whether to persist this task's configuration.
	Snapshot bool `json:"snapshot" msgpack:"snapshot"`
	// Status is the current execution status of the task.
	Status *Status `json:"status,omitempty" msgpack:"status,omitempty"`
}

// Command is a command to execute on a task in the Driver system.
type Command struct {
	// Task is the key of the target task.
	Task Key `json:"task" msgpack:"task"`
	// Type is the command type (e.g., 'start', 'stop', 'configure').
	Type string `json:"type" msgpack:"type"`
	// Key is a unique identifier for this command instance.
	Key string `json:"key" msgpack:"key"`
	// Args contains optional arguments for the command.
	Args msgpack.EncodedJSON `json:"args" msgpack:"args"`
}

var _ gorp.Entry[Key] = Task{}

func (t Task) GorpKey() Key      { return t.Key }
func (t Task) SetOptions() []any { return nil }

func OntologyID(k Key) ontology.ID {
	return ontology.ID{Type: ontology.ResourceTypeTask, Key: k.String()}
}
