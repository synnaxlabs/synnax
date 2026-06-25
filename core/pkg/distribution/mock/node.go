// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package mock

import (
	"context"

	"github.com/synnaxlabs/synnax/pkg/distribution"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/testutil"
)

// Node represents a single node in a cluster. It contains both the distribution layer
// it uses to talk to other nodes and the storage layer it uses to persist data.
type Node struct {
	// Layer is the distribution layer for the node. It routes channel and frames to the
	// appropriate nodes in the cluster.
	*distribution.Layer
	// Storage is the storage layer for the node. It persists frame data to this node.
	Storage *storage.Layer
	owner   *Cluster
}

// Close closes the node's distribution layer. For a Node returned by OpenNode it also
// closes the underlying single-node cluster's storage. This intentionally shadows the
// embedded Layer.Close so that OpenNode's caller can tear everything down through the
// returned node.
func (n Node) Close() error {
	err := n.Layer.Close()
	if n.owner != nil {
		err = errors.Join(err, n.owner.storage.Close())
	}
	return err
}

// MustOpenNode opens a single-node in-memory cluster and registers its teardown via
// testutil.DeferClose, returning the node. Like MustOpenCluster, it must be called from
// within a Ginkgo node; use OpenNode from plain Go tests and benchmarks.
func MustOpenNode(ctx context.Context) Node {
	return testutil.DeferClose(OpenNode(ctx))
}

// OpenNode opens a single-node in-memory cluster and returns its node. The caller owns
// teardown: closing the returned node tears down the whole cluster, including its
// storage.
func OpenNode(ctx context.Context) Node {
	c := OpenCluster(ctx, 1)
	n := c.Nodes[node.KeyBootstrapper]
	n.owner = c
	return n
}
