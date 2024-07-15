/*
 * Copyright 2024 Synnax Labs, Inc.
 *
 * Use of this software is governed by the Business Source License included in the file
 * licenses/BSL.txt.
 *
 * As of the Change Date specified in that file, in accordance with the Business Source
 * License, use of this software will be governed by the Apache License, Version 2.0,
 * included in the file licenses/APL.txt.
 */

package device

import (
	"github.com/synnaxlabs/synnax/pkg/hardware/rack"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/validate"
)

type Device struct {
	// The key of the device is its serial no.
	Key string `json:"key" msgpack:"key"`
	// Rack is the rack that the device is in.
	Rack rack.Key `json:"rack" msgpack:"rack"`
	// Location is the location of the device in the rack.
	Location string `json:"location" msgpack:"location"`
	// Name is a human-readable name for the device.
	Name string `json:"name" msgpack:"name"`
	// Identifier is a short ID that is useful for identifying items related to this
	// device.
	Identifier string `json:"identifier" msgpack:"identifier"`
	// Make is the manufacturer of the device.
	Make string `json:"make" msgpack:"make"`
	// Model is the model of the device.
	Model string `json:"model" msgpack:"model"`
	// Configured sets whether the device has been configured yet.
	Configured bool `json:"configured" msgpack:"configured"`
	// Properties
	Properties string `json:"properties" msgpack:"properties"`
}

var _ gorp.Entry[string] = Device{}

// GorpKey implements gorp.Entry.
func (d Device) GorpKey() string { return d.Key }

// SetOptions implements gorp.Entry.
func (d Device) SetOptions() []interface{} { return nil }

// Validate validates the device for creation.
func (d Device) Validate() error {
	v := validate.New("hardware.device")
	validate.NonZero(v, "rack", d.Rack)
	validate.NotEmptyString(v, "location", d.Location)
	validate.NotEmptyString(v, "name", d.Name)
	return v.Error()
}
