// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package versions

import v0 "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v0"

type (
	// Key is the cluster-unique identifier for a channel.
	Key = v0.Key
	// Keys is a slice of Key with convenience methods.
	Keys = v0.Keys
	// LocalKey is the 20-bit, node-local portion of a channel Key.
	LocalKey = v0.LocalKey
)
