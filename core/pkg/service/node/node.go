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

// Re-exports of the cluster-membership primitives defined in pkg/distribution/node.
// Consumers in the service and API layers should reach for these aliases so they only
// depend on a single node package.
type (
	// Node is a single Core of a Synnax cluster, identified by a Key and reachable at a
	// network address.
	Node = node.Node
	// Key is a 12-bit unsigned integer that uniquely identifies a Node within a
	// cluster.
	Key = node.Key
	// Change describes a single mutation to a Node's record.
	Change = node.Change
	// Cluster is the cluster-membership view as observed by the host node.
	Cluster = node.Cluster
	// ClusterChange is a batch of node-level Changes emitted by Cluster.OnChange.
	ClusterChange = node.ClusterChange
	// HostProvider exposes information about the host node without performing remote
	// resolution.
	HostProvider = node.HostProvider
)

const (
	// KeyFree is the reserved Key used for resources that are not leased to any
	// specific node — most notably free (virtual / non-persisted) channels. It is never
	// assigned to a real node.
	KeyFree = node.KeyFree
)
