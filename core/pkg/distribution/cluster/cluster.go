// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package cluster names a Core. The node domain type lives in pkg/distribution/node.
package cluster

import (
	"uuid"

	"github.com/synnaxlabs/synnax/pkg/distribution/node"
)

// Identity names a Core: the cluster it belongs to and its own node in that cluster.
type Identity struct {
	// Key is the key of the cluster. Clients use it to tell Cores apart.
	Key uuid.UUID
	// HostKey is the key of the host node. Channel, rack, and task keys embed it.
	HostKey node.Key
}
