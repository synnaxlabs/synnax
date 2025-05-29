// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package device

import (
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/hardware/rack"
	"github.com/synnaxlabs/x/gorp"
	xjson "github.com/synnaxlabs/x/json"
	"github.com/synnaxlabs/x/status"
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
	// Make is the manufacturer of the device.
	Make string `json:"make" msgpack:"make"`
	// Model is the model of the device.
	Model string `json:"model" msgpack:"model"`
	// Configured sets whether the device has been configured yet.
	Configured bool `json:"configured" msgpack:"configured"`
	// Properties
	Properties string `json:"properties" msgpack:"properties"`
	// State is the state of the device. This field is not stored directly with the
	// device inside of gorp, and is not guaranteed to be valid.
	State *State `json:"state" msgpack:"state"`
}

var _ gorp.Entry[string] = Device{}

// GorpKey implements gorp.Entry.
func (d Device) GorpKey() string { return d.Key }

// SetOptions implements gorp.Entry.
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

// State represents the state of a device.
type State struct {
	// Key is the key of the device.
	Key string `json:"key" msgpack:"key"`
	// Rack is the rack that the device is in.
	Rack rack.Key `json:"rack" msgpack:"rack"`
	// Variant is the status variant representing the general state of the device.
	Variant status.Variant `json:"variant" msgpack:"variant"`
	// Details are JSON-stringified details about the device's state. These are arbitrary,
	// and vary based on the device vendor.
	Details xjson.String `json:"details" msgpack:"details"`
}

// GorpKey implements gorp.Entry.
func (s State) GorpKey() string { return s.Key }

// SetOptions implements gorp.Entry.
func (s State) SetOptions() []any { return []any{s.Rack.Node()} }
