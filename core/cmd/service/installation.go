// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package service provides the commands that run the Core as a Linux or Windows
// service, and the controls that `synnax update` uses on it.
package service

// Installation is the Core service on this host.
type Installation struct {
	// Binary is the path of the Core binary that the service runs.
	Binary string
	// Data is the data directory of the Core that the service runs.
	Data string
}
