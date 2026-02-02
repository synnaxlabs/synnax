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
	"fmt"

	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
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
func ParseDeviceProperties(s string) (DeviceProperties, error) {
	var p DeviceProperties
	if err := json.Unmarshal([]byte(s), &p); err != nil {
		return p, errors.Wrap(err, "failed to parse Sift device properties")
	}
	return p, nil
}

// TaskConfig contains all parameters for a Sift upload task.
type TaskConfig struct {
	// DeviceKey references the Sift device containing connection config.
	DeviceKey string `json:"device_key"`
	// AssetName is the Sift asset name to upload to.
	AssetName string `json:"asset_name"`
	// FlowName is the Sift flow name for this upload.
	FlowName string `json:"flow_name"`
	// RunName is the Sift run name. A run will be created with this name.
	RunName string `json:"run_name"`
	// Channels are the Synnax channel keys to upload.
	Channels []channel.Key `json:"channels"`
	// TimeRange is the time range to upload.
	TimeRange telem.TimeRange `json:"time_range"`
}

// ParseTaskConfig parses TaskConfig from a JSON string.
func ParseTaskConfig(s string) (TaskConfig, error) {
	var c TaskConfig
	if err := json.Unmarshal([]byte(s), &c); err != nil {
		return c, errors.Wrap(err, "failed to parse Sift task config")
	}
	return c, nil
}

// clientKey generates a deterministic client key from the task key for ingestion
// config identification.
func clientKey(taskKey task.Key) string {
	return fmt.Sprintf("synnax-task-%d", taskKey)
}
