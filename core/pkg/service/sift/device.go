// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package sift

import (
	"encoding/json"

	"github.com/synnaxlabs/x/errors"
)

const (
	// DeviceMake is the device make identifier for Sift devices.
	DeviceMake = "Sift"
	// DeviceModel is the device model identifier.
	DeviceModel = "Cloud"
)

// DeviceProperties contains the Sift connection configuration stored in
// device.Properties.
type DeviceProperties struct {
	// URI is the Sift API endpoint (e.g., "api.siftstack.com:443").
	URI string `json:"uri"`
	// APIKey is the Sift API key for authentication.
	APIKey string `json:"api_key"`
}

// ParseDeviceProperties parses DeviceProperties from a JSON string.
func ParseDeviceProperties(properties string) (DeviceProperties, error) {
	var p DeviceProperties
	if err := json.Unmarshal([]byte(properties), &p); err != nil {
		return p, errors.Wrap(err, "failed to parse device properties")
	}
	return p, nil
}
