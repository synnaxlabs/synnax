// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package device implements types and services for managing physical pieces of hardware
// in Synnax. This includes creating, retrieving, and updating devices.
package device

import (
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/validate"
)

// Device represents a unique piece of physical hardware that is connected to a rack.
// Examples of devices include DAQ cards, PLCs, and other hardware.
type Device struct {
	// The key of the device is its serial no.
	Key string `json:"key" msgpack:"key"`
	// Rack is the rack that the device is in.
	Rack rack.Key `json:"rack" msgpack:"rack"`
	// Location is the location of the device in the rack.
	Location string `json:"location" msgpack:"location"`
	// Name is a human-readable name for the device.
	Name string `json:"name" msgpack:"name"`
	// Make is the manufacturer of the device.
	Make string `json:"make" msgpack:"make"`
	// Model is the model of the device.
	Model string `json:"model" msgpack:"model"`
	// Configured sets whether the device has been configured yet.
	Configured bool `json:"configured" msgpack:"configured"`
	// Properties are additional properties that are unique to the device.
	Properties string `json:"properties" msgpack:"properties"`
	// ParentDevice is the key of the parent device, if this device is a child of another
	// device (e.g., a module in a chassis). If empty, the device is a root device and
	// its parent in the ontology is the rack.
	ParentDevice string `json:"parent_device,omitempty" msgpack:"parent_device"`
	// Status is the state of the device. This field is not stored directly with the
	// device inside of gorp, and is not guaranteed to be valid.
	Status *Status `json:"status" msgpack:"status"`
}

var _ gorp.Entry[string] = Device{}

// GorpKey gives a unique key for the device for use in gorp.
func (d Device) GorpKey() string { return d.Key }

// SetOptions returns nil.
func (d Device) SetOptions() []any { return nil }

// OntologyID returns the unique ID for the device within the ontology.
func (d Device) OntologyID() ontology.ID { return OntologyID(d.Key) }

// Validate validates the device for creation.
func (d Device) Validate() error {
	v := validate.New("hardware.device")
	validate.NonZero(v, "rack", d.Rack)
	validate.NotEmptyString(v, "location", d.Location)
	validate.NotEmptyString(v, "name", d.Name)
	return v.Error()
}

// StatusDetails represents unique information about the device's status that appears in
// the status payload.
type StatusDetails struct {
	// Rack identifies the rack that the device is currently connected to.
	Rack rack.Key `json:"rack" msgpack:"rack"`
	// Device identifies the key of the device that this status is for.
	Device string `json:"device" msgpack:"device"`
}

// Status represents information about the state of the device at a given point in time.
type Status = status.Status[StatusDetails]
