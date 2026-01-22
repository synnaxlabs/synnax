// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package sift provides integration with Sift for uploading historical data
// and streaming real-time telemetry.
package sift

const (
	// TaskTypeUploader is the task type for historical data upload.
	TaskTypeUploader = "sift_upload"
	// TaskTypeWriter is the task type for real-time data streaming.
	TaskTypeWriter = "sift_write"
	// DeviceMake is the device make identifier for Sift devices.
	DeviceMake = "Sift"
	// DeviceModel is the device model identifier for Sift devices.
	DeviceModel = "Cloud"
)
