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

import "github.com/synnaxlabs/x/gorp"

type Device struct {
	// The key of the device is its serial no.
	Key string `json:"key" msgpack:"key"`
	// Name is a human-readable name for the device.
	Name string `json:"name" msgpack:"name"`
	// Identifier is a short ID that is useful for identifying items related to this
	// device.
	Identifier string `json:"identifier" msgpack:"identifier"`
	// Make is the manufacturer of the device.
	Make string `json:"make" msgpack:"make"`
	// Model is the model of the device.
	Model string `json:"model" msgpack:"model"`
	// Properties
	Properties string `json:"properties" msgpack:"properties"`
}

var _ gorp.Entry[string] = Device{}

// GorpKey implements gorp.Entry.
func (d Device) GorpKey() string { return d.Key }

// SetOptions implements gorp.Entry.
func (d Device) SetOptions() []interface{} { return nil }
