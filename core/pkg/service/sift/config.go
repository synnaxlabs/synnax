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
}

// ParseDeviceProperties parses DeviceProperties from a JSON string.
func ParseDeviceProperties(s string) (DeviceProperties, error) {
	var p DeviceProperties
	if err := json.Unmarshal([]byte(s), &p); err != nil {
		return p, errors.Wrap(err, "failed to parse Sift device properties")
	}
	return p, nil
}

// clientKey generates a deterministic client key for ingestion config deduplication.
func (p DeviceProperties) clientKey() string {
	// FNV-1a hash for deterministic key generation
	var hash uint64 = 14695981039346656037
	s := p.URI + ":" + p.AssetName
	for i := 0; i < len(s); i++ {
		hash ^= uint64(s[i])
		hash *= 1099511628211
	}
	return "synnax-" + string(rune(hash))
}

// TaskConfig contains all parameters for a Sift upload task. Each task represents a
// single upload operation and is deleted upon completion.
type TaskConfig struct {
	// DeviceKey references the Sift device containing connection config.
	DeviceKey string `json:"device_key"`
	// RangeKey is the Synnax range to upload.
	RangeKey uuid.UUID `json:"range_key"`
	// Channels are the Synnax channel keys to upload.
	Channels []channel.Key `json:"channels"`
	// FlowName is the Sift flow name for this upload.
	FlowName string `json:"flow_name"`
	// RunName is an optional Sift run name. If provided, a run will be created.
	RunName string `json:"run_name,omitempty"`
	// TimeRange optionally overrides the range time bounds.
	TimeRange *telem.TimeRange `json:"time_range,omitempty"`
}
