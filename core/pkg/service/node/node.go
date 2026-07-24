// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package node

import (
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
)

// Re-exports of the node domain primitives defined in pkg/distribution/node. Consumers
// in the service and API layers should reach for these aliases so they only depend on a
// single node package.
type (
	// Node is a single Core of a Synnax cluster, identified by a Key and reachable at a
	// network address.
	Node = node.Node
	// Change describes a single mutation to a Node's record.
	Change = node.Change
)

const (
	// KeyBootstrapper is the reserved Key assigned to the first node in a cluster (the
	// node that bootstraps a new cluster rather than joining an existing one).
	KeyBootstrapper = node.KeyBootstrapper
	// KeyFree is the reserved Key used for resources that are not leased to any
	// specific node — most notably free (virtual / non-persisted) channels. It is never
	// assigned to a real node.
	KeyFree = node.KeyFree
)
