// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package node exposes the node domain type over aspen's membership primitives. The
// cluster-membership view lives in pkg/distribution/cluster; the ontology service that
// publishes nodes as resources lives in pkg/service/node.
package node

import "github.com/synnaxlabs/aspen"

type (
	// Key is a 12-bit unsigned integer that uniquely identifies a Node within a
	// cluster. Keys are assigned dynamically when a node joins the cluster (via
	// distributed counter through aspen's pledge protocol). Two reserved values are
	// defined: KeyFree and KeyBootstrapper.
	Key = aspen.NodeKey
	// Node is a single Core of a Synnax cluster, identified by a Key and reachable at a
	// network address. Nodes report a State that reflects their current reachability as
	// observed by the host Core.
	Node = aspen.Node
	// Change describes a single mutation to a Node's record (a node joining, leaving,
	// or transitioning state). Changes are emitted by the cluster-membership view as
	// part of a cluster.Change batch.
	Change = aspen.NodeChange
)

const (
	// KeyBootstrapper is the reserved Key assigned to the first node in a cluster (the
	// node that bootstraps a new cluster rather than joining an existing one).
	KeyBootstrapper = aspen.NodeKeyBootstrapper
	// KeyFree is the reserved Key used for resources that are not leased to any
	// specific node — most notably free (virtual / non-persisted) channels. It is never
	// assigned to a real node.
	KeyFree = aspen.NodeKeyFree
)
