// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0

import (
	channel "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
)

// Version is the ImEx schema version of log data at this state. The Console stamped it
// on the wire as the semver string "0.0.0", which legacy.MigrateData decodes onto this
// numeric version.
const Version imex.Version = 0

// Data is the frozen type for log data at version 0. Channels are stored as bare
// channel keys. Key, Name, Type, and Version are envelope-level fields and are not part
// of Data.
type Data struct {
	Channels      []channel.Key `json:"channels"`
	RemoteCreated bool          `json:"remoteCreated"`
}
