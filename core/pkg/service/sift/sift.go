// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package sift provides integration with Sift for uploading historical data.
package sift

import (
	"github.com/synnaxlabs/synnax/pkg/service/sift/client"
	"github.com/synnaxlabs/synnax/pkg/service/sift/device"
	"github.com/synnaxlabs/synnax/pkg/service/sift/upload"
)

// ClientFactory is the default client factory for creating Sift gRPC clients.
var ClientFactory = client.NewGRPC

// Re-export device constants for backward compatibility.
const (
	DeviceMake  = device.Make
	DeviceModel = device.Model
)

// DeviceProperties is an alias for device.Properties.
type DeviceProperties = device.Properties

// UploadTaskType is the task type for Sift uploads.
const UploadTaskType = upload.TaskType

// UploadTaskConfig is the configuration for an upload task.
type UploadTaskConfig = upload.Config
