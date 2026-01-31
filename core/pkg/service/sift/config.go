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

	"github.com/google/uuid"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
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
	// AssetName is the Sift asset name to associate data with.
	AssetName string `json:"asset_name"`
	// ClientKey is a unique identifier for ingestion config deduplication.
	ClientKey string `json:"client_key"`
	// OrganizationID is an optional organization identifier.
	OrganizationID string `json:"organization_id,omitempty"`
}

// ParseDeviceProperties parses DeviceProperties from a JSON string.
func ParseDeviceProperties(s string) (DeviceProperties, error) {
	var p DeviceProperties
	if err := json.Unmarshal([]byte(s), &p); err != nil {
		return p, errors.Wrap(err, "failed to parse Sift device properties")
	}
	return p, nil
}

// UploaderTaskConfig is the configuration for a Sift uploader task. The uploader task
// is auto-created on boot for each Sift device. Channels come from upload commands, not
// from the task config.
type UploaderTaskConfig struct {
	// DeviceKey references the Sift device containing connection config.
	DeviceKey string `json:"device_key"`
}

// UploadCommand is the command sent to trigger a historical data upload.
type UploadCommand struct {
	// RangeKey is the Synnax range to upload.
	RangeKey uuid.UUID `json:"range_key"`
	// Channels are the Synnax channel keys to upload.
	Channels []channel.Key `json:"channels"`
	// FlowName is the Sift flow name for this upload.
	FlowName string `json:"flow_name"`
	// RunName is an optional Sift run name.
	RunName string `json:"run_name,omitempty"`
	// TimeRange is an optional time range override (uses range bounds if not set).
	TimeRange *telem.TimeRange `json:"time_range,omitempty"`
	// ChunkSize is an optional chunk size for uploads.
	ChunkSize int64 `json:"chunk_size,omitempty"`
}
