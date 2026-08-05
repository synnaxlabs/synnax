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

	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	rack "github.com/synnaxlabs/synnax/pkg/service/rack/versions/v0"
	status "github.com/synnaxlabs/synnax/pkg/service/status/versions/v0"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/gorp"
)

// Key is a composite identifier for a task. The high 32 bits contain the rack key, and
// the low 32 bits contain the local task key within that rack.
type Key uint64

// String returns the key formatted as its decimal integer value.
func (k Key) String() string { return strconv.FormatUint(uint64(k), 10) }

// Rack returns the key of the rack this task belongs to.
func (k Key) Rack() rack.Key { return rack.Key(k >> 32) }

// LocalKey returns the task's unique key within its rack.
func (k Key) LocalKey() uint32 { return uint32(uint64(k) & 0xFFFFFFFF) }

// IsValid returns true when both the rack and local components are set.
func (k Key) IsValid() bool { return !k.Rack().IsZero() && k.LocalKey() != 0 }

// OntologyID returns the unique ontology identifier for the task.
func (k Key) OntologyID() ontology.ID {
	return ontology.ID{Type: ontology.ResourceTypeTask, Key: k.String()}
}

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
	Data msgpack.EncodedJSON `json:"data,omitzero" msgpack:"data,omitzero"`
}

// Task is an executable unit of work in the Driver system. Tasks represent specific
// hardware operations such as reading sensor data, writing control signals, or scanning
// for devices.
type Task struct {
	// Key is the composite identifier for this task.
	Key Key `json:"key" msgpack:"key"`
	// Name is a human-readable name for the task.
	Name string `json:"name" msgpack:"name"`
	// Type is the task type (e.g., 'modbus_read', 'labjack_write', 'opc_scan').
	// Determines which hardware integration handles the task.
	Type string `json:"type" msgpack:"type"`
	// Config is task-specific configuration stored as JSON. Structure varies by task
	// type.
	Config msgpack.EncodedJSON `json:"config" msgpack:"config"`
	// Internal is true if this is an internal system task.
	Internal bool `json:"internal" msgpack:"internal"`
	// Snapshot indicates whether to persist this task's configuration.
	Snapshot bool `json:"snapshot" msgpack:"snapshot"`
	// Status is the current execution status of the task.
	Status *Status `json:"status,omitempty" msgpack:"status,omitempty"`
}

var _ gorp.Entry[Key] = Task{}

// GorpKey implements gorp.Entry.
func (t Task) GorpKey() Key { return t.Key }

// SetOptions implements gorp.Entry.
func (Task) SetOptions() []any { return nil }

// OntologyID returns the unique ontology identifier for the task.
func (t Task) OntologyID() ontology.ID { return t.Key.OntologyID() }
