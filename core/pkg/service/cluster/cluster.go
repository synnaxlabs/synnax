// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cluster

import "github.com/synnaxlabs/synnax/pkg/distribution/cluster"

// Re-exports of the cluster-membership primitives defined in pkg/distribution/cluster.
// Consumers in the service and API layers should reach for these aliases so they only
// depend on a single cluster package.
type (
	// Cluster is the cluster-membership view as observed by the host node.
	Cluster = cluster.Cluster
	// Change is a batch of node-level changes emitted by Cluster.OnChange.
	Change = cluster.Change
	// HostProvider exposes information about the host node without performing remote
	// resolution.
	HostProvider = cluster.HostProvider
	// HostResolver resolves node keys to network addresses and exposes information
	// about the host node.
	HostResolver = cluster.HostResolver
)
