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
	"github.com/synnaxlabs/x/signal"
)

type (
	Transport    = transport.Transport
	Cluster      = cluster.Cluster
	Resolver     = cluster.Resolver
	HostResolver = cluster.HostResolver
	Node         = node.Node
	NodeID       = node.ID
	Address      = address.Address
	NodeState    = node.State
	ClusterState = cluster.State
)

const (
	Healthy = node.StateHealthy
	Left    = node.StateLeft
	Dead    = node.StateDead
	Suspect = node.StateSuspect
)

type DB interface {
	Cluster
	kvx.DB
}

type db struct {
	Cluster
	kvx.DB
	transport struct {
		shutdown context.CancelFunc
		wg       signal.WaitGroup
	}
}

func (db *db) Close() error {
	db.transport.shutdown()
	c := errutil.NewCatch(errutil.WithAggregation())
	c.Exec(db.transport.wg.Wait)
	c.Exec(db.Cluster.Close)
	c.Exec(db.DB.Close)
	return lo.Ternary(errors.Is(c.Error(), context.Canceled), nil, c.Error())
}
