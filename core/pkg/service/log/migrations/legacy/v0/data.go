// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0

import "github.com/synnaxlabs/synnax/pkg/distribution/channel"

// Version is the semantic version string written by the console at this state version.
const Version = "0.0.0"

// Data is the frozen type for log data at version 0. Channels are stored as bare
// channel keys. Key, Name, Type, and Version are envelope-level fields and are not part
// of Data.
type Data struct {
	Channels      []channel.Key `json:"channels"`
	RemoteCreated bool          `json:"remoteCreated"`
}
