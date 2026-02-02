// Copyright 2026 Synnax Labs, Inc.
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

	"github.com/synnaxlabs/x/errors"
)

const (
	// Make is the device make identifier for Sift devices.
	Make = "sift"
	// Model is the device model identifier.
	Model = "cloud"
)

// Properties contains the Sift connection configuration stored in device.Properties.
type Properties struct {
	// APIKey is the Sift API key for authentication.
	APIKey string `json:"api_key"`
	// URI is the Sift gRPC API endpoint (e.g., "grpc-api.siftstack.com:443"). This
	// should not include the protocol (e.g., "https://"), and should include the port
	// (e.g., ":443").
	URI string `json:"uri"`
}

// ParseProperties parses Sift device properties from a JSON string.
func ParseProperties(properties string) (Properties, error) {
	var p Properties
	if err := json.Unmarshal([]byte(properties), &p); err != nil {
		return p, errors.Wrap(err, "failed to parse device properties")
	}
	return p, nil
}
