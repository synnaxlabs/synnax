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
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	rack "github.com/synnaxlabs/synnax/pkg/service/rack/migrations/v0"
	status "github.com/synnaxlabs/synnax/pkg/service/status/migrations/v0"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/gorp"
)

// Status is device-specific status information including operational state and device
// identification.
type Status = status.Status[StatusDetails]

// Key is the device's serial number, used as its unique identifier.
type Key = string

// StatusDetails contains device-specific status details identifying the device and its
// associated rack.
type StatusDetails struct {
	// Rack is the key of the rack this device belongs to.
	Rack rack.Key `json:"rack" msgpack:"rack"`
	// Device is the device identifier.
	Device string `json:"device" msgpack:"device"`
}

// Device is a physical piece of hardware connected to Synnax through the Driver system.
// Devices represent external equipment like LabJack, National Instruments, OPC UA
// servers, or Modbus devices.
type Device struct {
	// Key is the unique identifier for this device.
	Key Key `json:"key" msgpack:"key"`
	// Rack is the key of the rack that owns this device.
	Rack rack.Key `json:"rack" msgpack:"rack"`
	// Location is the physical location or address of the device.
	Location string `json:"location" msgpack:"location"`
	// Make is the manufacturer of the device (e.g., 'LabJack', 'National Instruments').
	Make string `json:"make" msgpack:"make"`
	// Model is the device model identifier.
	Model string `json:"model" msgpack:"model"`
	// Name is a human-readable name for the device.
	Name string `json:"name" msgpack:"name"`
	// Configured indicates whether the device has been successfully configured and is ready
	// for use.
	Configured bool `json:"configured" msgpack:"configured"`
	// Properties contains device-specific configuration properties stored as JSON.
	// Structure varies by device make and model.
	Properties msgpack.EncodedJSON `json:"properties" msgpack:"properties"`
	// Status is the current operational status of the device.
	Status *Status `json:"status,omitempty" msgpack:"status,omitempty"`
	// Parent is an optional parent resource ID for hierarchical device organization (e.g.,
	// NI chassis containing modules).
	Parent *ontology.ID `json:"parent,omitempty" msgpack:"parent,omitempty"`
}

var _ gorp.Entry[Key] = Device{}

func (d Device) GorpKey() Key      { return d.Key }
func (d Device) SetOptions() []any { return nil }

func OntologyID(key string) ontology.ID {
	return ontology.ID{Type: ontology.ResourceTypeDevice, Key: key}
}
