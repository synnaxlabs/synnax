// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package node

import "github.com/synnaxlabs/synnax/pkg/distribution/node"

// Re-exports of the node domain primitives defined in pkg/distribution/node. Consumers
// in the service and API layers should reach for these aliases so they only depend on a
// single node package.
type (
	// Node is a node of a Core, reached at its address.
	Node = node.Node
)

const (
	// KeyFree is the key of no node. Resources that no node leases, such as free
	// virtual channels, carry it.
	KeyFree = node.KeyFree
)
