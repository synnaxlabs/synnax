// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package node exposes a thin domain layer over aspen's cluster-membership primitives,
// along with the ontology service that publishes nodes as resources in the Synnax
// ontology.
package node

import "github.com/synnaxlabs/aspen"

type (
	// Node is a single Core of a Synnax cluster, identified by a Key and reachable at a
	// network address. Nodes report a State that reflects their current reachability as
	// observed by the host Core.
	Node = aspen.Node
	// Key is a 12-bit unsigned integer that uniquely identifies a Node within a
	// cluster. Keys are assigned dynamically when a node joins the cluster (via
	// distributed counter through aspen's pledge protocol). Two reserved values are
	// defined: KeyFree and KeyBootstrapper.
	Key = aspen.NodeKey
	// State describes the reachability of a Node from the host's perspective. The
	// concrete values (Healthy, Suspect, Dead, Left) live on aspen's node package and
	// propagate through the cluster via SI gossip.
	State = aspen.NodeState
	// Change describes a single mutation to a Node's record (a node joining, leaving,
	// or transitioning state). Changes are emitted by Cluster.OnChange as part of a
	// ClusterChange batch.
	Change = aspen.NodeChange
	// Cluster is the cluster-membership view as observed by the host node. It resolves
	// node addresses, exposes the current set of Nodes, and emits ClusterChange events
	// as topology evolves. Cluster is safe for concurrent use.
	Cluster = aspen.Cluster
	// ClusterChange is a batch of node-level Changes emitted by Cluster.OnChange when
	// the gossip layer integrates new state from a peer.
	ClusterChange = aspen.ClusterChange
	// ClusterState is a point-in-time snapshot of the cluster as the host sees it,
	// including the host's own Key and the Group of all known Nodes.
	ClusterState = aspen.ClusterState
	// Resolver maps a Key to the network Address at which that node can be reached.
	Resolver = aspen.Resolver
	// HostResolver is a Resolver that also exposes information about the host node it
	// is running on (its Key and full Node record).
	HostResolver = aspen.HostResolver
	// HostProvider exposes information about the host node — its Key and full Node
	// record — without performing remote resolution.
	HostProvider = aspen.HostProvider
)

const (
	// KeyFree is the reserved Key used for resources that are not leased to any
	// specific node — most notably free (virtual / non-persisted) channels. It is never
	// assigned to a real node.
	KeyFree = aspen.NodeKeyFree
	// KeyBootstrapper is the reserved Key assigned to the first node in a cluster (the
	// node that bootstraps a new cluster rather than joining an existing one).
	KeyBootstrapper = aspen.NodeKeyBootstrapper
)
