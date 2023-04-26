// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package aspen

import (
	"context"
	"github.com/cockroachdb/errors"
	"github.com/samber/lo"
	"github.com/synnaxlabs/aspen/internal/cluster"
	"github.com/synnaxlabs/aspen/internal/node"
	"github.com/synnaxlabs/aspen/transport"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/errutil"
	kvx "github.com/synnaxlabs/x/kv"
	"io"
)

type (
	Transport     = transport.Transport
	Cluster       = cluster.Cluster
	Resolver      = cluster.Resolver
	HostResolver  = cluster.HostResolver
	Node          = node.Node
	NodeKey       = node.Key
	NodeChange    = node.Change
	Address       = address.Address
	NodeState     = node.State
	ClusterState  = cluster.State
	ClusterChange = cluster.Change
)

const (
	Healthy = node.StateHealthy
	Left    = node.StateLeft
	Dead    = node.StateDead
	Suspect = node.StateSuspect
)

type DB struct {
	Cluster Cluster
	kvx.DB
	transportShutdown io.Closer
}

// Close implements kvx.DB, shutting down the key-value store, cluster and transport.
// Close is not safe to call concurrently with any other DB method. All DB methods
// called after Close will panic.
func (db *DB) Close() error {
	c := errutil.NewCatch(errutil.WithAggregation())
	c.Exec(db.transportShutdown.Close)
	c.Exec(db.Cluster.Close)
	c.Exec(db.DB.Close)
	return lo.Ternary(errors.Is(c.Error(), context.Canceled), nil, c.Error())
}
