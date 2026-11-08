// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package testutil

import (
	"context"

	"github.com/onsi/ginkgo/v2"
	"github.com/synnaxlabs/synnax/pkg/distribution"
	"github.com/synnaxlabs/synnax/pkg/storage"
	storagetestutil "github.com/synnaxlabs/synnax/pkg/storage/testutil"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/testutil"
)

// Node is a memory-backed Core: a distribution layer and the storage layer it persists
// to.
type Node struct {
	// Layer is the distribution layer of the node.
	*distribution.Layer
	// Storage is the storage layer of the node.
	Storage *storage.Layer
	// storageOwned is true when the Node opened Storage.
	storageOwned bool
}

// Close closes the distribution layer, then the storage layer if the Node opened it. It
// shadows the embedded Layer.Close.
func (n Node) Close() error {
	err := n.Layer.Close()
	if n.storageOwned {
		err = errors.Join(err, n.Storage.Close())
	}
	return err
}

// NewNode opens a Node and registers its teardown via testutil.DeferClose. It must be
// called from within a Ginkgo node; use OpenNode from plain Go tests and benchmarks.
func NewNode(ctx context.Context, overrides ...distribution.LayerConfig) Node {
	ginkgo.GinkgoHelper()
	return testutil.DeferClose(OpenNode(ctx, overrides...))
}

// OpenNode opens a Node. The overrides are layered on top of the base
// distribution.LayerConfig. The Node opens its own storage layer unless an override
// passes one, which the caller then closes. The caller closes the returned Node.
func OpenNode(ctx context.Context, overrides ...distribution.LayerConfig) Node {
	ginkgo.GinkgoHelper()
	var n Node
	for _, o := range overrides {
		if o.Storage != nil {
			n.Storage = o.Storage
		}
	}
	if n.Storage == nil {
		n.Storage = storagetestutil.OpenLayer(ctx)
		n.storageOwned = true
	}
	addr := address.Address("localhost:0")
	cfg := distribution.LayerConfig{Storage: n.Storage, AdvertiseAddress: addr}
	cfgs := append([]distribution.LayerConfig{cfg}, overrides...)
	n.Layer = testutil.MustSucceed(distribution.OpenLayer(ctx, cfgs...))
	n.Framer.SetFreeIndexResolver(nopFreeIndexResolver{})
	return n
}
