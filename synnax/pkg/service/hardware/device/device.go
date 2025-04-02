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
	"encoding/json"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/hardware/rack"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/errors"
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
	// Make is the manufacturer of the device.
	Make string `json:"make" msgpack:"make"`
	// Model is the model of the device.
	Model string `json:"model" msgpack:"model"`
	// Configured sets whether the device has been configured yet.
	Configured bool `json:"configured" msgpack:"configured"`
	// Properties
	Properties string `json:"properties" msgpack:"properties"`
	State      State  `json:"state" msgpack:"state"`
}

var _ gorp.Entry[string] = Device{}

// GorpKey implements gorp.Entry.
func (d Device) GorpKey() string { return d.Key }

// SetOptions implements gorp.Entry.
func (d Device) SetOptions() []interface{} { return nil }

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

type Details string

var detailsCodec = &binary.JSONCodec{}

func NewStaticDetails(data interface{}) Details {
	b, err := detailsCodec.Encode(nil, data)
	if err != nil {
		panic(err)
	}
	return Details(b)
}

// UnmarshalJSON implements the json.Unmarshaler interface for Details.
// It should correctly handle a raw JSON string or a JSON object/array.
func (d *Details) UnmarshalJSON(data []byte) error {
	// Try to unmarshal data into a plain string
	var plainString string
	if err := json.Unmarshal(data, &plainString); err == nil {
		*d = Details(plainString)
		return nil
	}

	// If the above fails, it means the data might be an object or an array,
	// so we re-marshal it into a string regardless of its type.
	var obj interface{}
	if err := json.Unmarshal(data, &obj); err != nil {
		return errors.New("input data is neither a plain string nor valid JSON")
	}

	// Marshal the object back to string
	bytes, err := json.Marshal(obj)
	if err != nil {
		return err
	}

	*d = Details(bytes)
	return nil
}

type Status string

// State represents the state of a device.
type State struct {
	Key     string   `json:"key" msgpack:"key"`
	Rack    rack.Key `json:"rack" msgpack:"rack"`
	Variant string   `json:"variant" msgpack:"variant"`
	Details Details  `json:"details" msgpack:"details"`
}

func (s State) GorpKey() string { return s.Key }

func (s State) SetOptions() []interface{} { return []interface{}{s.Rack.Node()} }
