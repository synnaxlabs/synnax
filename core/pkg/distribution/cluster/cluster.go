// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package cluster exposes a thin domain layer over aspen's cluster-membership view.
// The node domain type itself lives in pkg/distribution/node.
package cluster

import "github.com/synnaxlabs/aspen"

type (
	// Cluster is the cluster-membership view as observed by the host node. It resolves
	// node addresses, exposes the current set of Nodes, and emits Change events as
	// topology evolves. Cluster is safe for concurrent use.
	Cluster = aspen.Cluster
	// Change is a batch of node-level changes emitted by Cluster.OnChange when the
	// gossip layer integrates new state from a peer.
	Change = aspen.ClusterChange
	// State is a point-in-time snapshot of the cluster as the host sees it, including
	// the host's own node key and the group of all known nodes.
	State = aspen.ClusterState
	// Resolver maps a node key to the network address at which that node can be
	// reached.
	Resolver = aspen.Resolver
	// HostResolver is a Resolver that also exposes information about the host node it
	// is running on (its key and full node record).
	HostResolver = aspen.HostResolver
	// HostProvider exposes information about the host node — its key and full node
	// record — without performing remote resolution.
	HostProvider = aspen.HostProvider
)
