// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package aspen

import (
	"github.com/google/uuid"
	"github.com/synnaxlabs/aspen/internal/cluster"
	"github.com/synnaxlabs/aspen/node"
	"github.com/synnaxlabs/aspen/transport"
	"github.com/synnaxlabs/x/address"
	xio "github.com/synnaxlabs/x/io"
	xkv "github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/observe"
	storex "github.com/synnaxlabs/x/store"
)

// Cluster represents a group of nodes that can exchange their state with each other.
type Cluster interface {
	HostResolver
	// Key returns a unique key for the cluster. This value is consistent across
	// all nodes in the cluster.
	Key() uuid.UUID
	// Nodes returns a node.Group of all nodes in the cluster. The returned map
	// is not safe to modify. To modify, use node.Group.CopyState().
	Nodes() node.Group
	// Node returns the member Node with the given Key.
	Node(id node.Key) (node.Node, error)
	// Observable can be used to monitor changes to the cluster state. Be careful not to modify the
	// contents of the returned State.
	observe.Observable[cluster.Change]
	// Reader allows reading the current state of the cluster.
	storex.Reader[cluster.State]
}

// Resolver is used to resolve a reachable address for a node in the cluster.
type Resolver interface {
	// Resolve resolves the address of a node with the given key.
	Resolve(key node.Key) (address.Address, error)
}

type HostProvider interface {
	// Host returns the host Node (i.e. the node that HostProvider is called on).
	Host() node.Node
	// HostKey returns the Name of the host node.
	HostKey() node.Key
}

type HostResolver interface {
	Resolver
	HostProvider
}

type (
	Transport     = transport.Transport
	Node          = node.Node
	NodeKey       = node.Key
	NodeChange    = node.Change
	Address       = address.Address
	NodeState     = node.State
	ClusterState  = cluster.State
	ClusterChange = cluster.Change
)

const (
	Free         = node.Free
	Bootstrapper = node.Bootstrapper
	Healthy      = node.StateHealthy
	Left         = node.StateLeft
	Dead         = node.StateDead
	Suspect      = node.StateSuspect
)

var NodeNotfound = cluster.NodeNotFound

type DB struct {
	Cluster *cluster.Cluster
	xkv.DB
	closer xio.MultiCloser
}

// Close implements xkv.DB, shutting down the key-value store, cluster and transport.
// Close is not safe to call concurrently with any other DB method. All DB methods
// called after Close will panic.
func (db *DB) Close() error { return db.closer.Close() }
