// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package cluster provides an interface for joining a cluster of nodes and
// exchanging state through an SI gossip model. nodes can join the cluster without
// needing to know all members. Cluster will automatically manage the membership of
// new nodes by assigning them unique IDs and keeping them in sync with their peers.
// To Open a cluster, simply use cluster.Open.
package cluster

import (
	"context"
	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/synnaxlabs/aspen/internal/cluster/gossip"
	pledge_ "github.com/synnaxlabs/aspen/internal/cluster/pledge"
	"github.com/synnaxlabs/aspen/internal/cluster/store"
	"github.com/synnaxlabs/aspen/internal/node"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/iter"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/signal"
	storex "github.com/synnaxlabs/x/store"
	"go.uber.org/zap"
	"io"
)

// State represents the current state of the cluster as seen from the host node.
type State = store.State

// NodeNotFound is returned when a node cannot be found in the cluster.
var NodeNotFound = errors.New("[cluster] - node not found")

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
	// Observable returns can be used to monitor changes to the cluster state. Be careful not to modify the
	// contents of the returned State.
	observe.Observable[State]
	// Reader allows reading the current state of the cluster.
	storex.Reader[State]
	io.Closer
}

// Resolver is used to resolve a reachable address for a node in the cluster.
type Resolver interface {
	// Resolve resolves the address of a node with the given Key.
	Resolve(key node.Key) (address.Address, error)
}

type Host interface {
	// Host returns the host Node (i.e. the node that Host is called on).
	Host() node.Node
	// HostKey returns the Key of the host node.
	HostKey() node.Key
}

type HostResolver interface {
	Resolver
	Host
}

// Open joins the host node to the cluster and begins gossiping its state. The
// node will spread addr as its listening address. A set of peer addresses
// (other nodes in the cluster) must be provided when joining an existing cluster
// for the first time. If restarting a node that is already a member of a cluster,
// the peer addresses can be left empty; Open will attempt to load the existing
// cluster state from storage (see Config.Storage and Config.StorageKey).
// If provisioning a new cluster, ensure that all storage for previous clusters
// is removed and provide no peers.
func Open(ctx context.Context, configs ...Config) (Cluster, error) {
	cfg, err := newConfig(ctx, configs)
	if err != nil {
		return nil, err
	}

	sCtx, cancel := signal.WithCancel(cfg.T.Transfer(ctx, context.Background()))

	c := &cluster{
		Store:    cfg.Gossip.Store,
		shutdown: signal.NewShutdown(sCtx, cancel),
		Config:   cfg,
	}

	// Attempt to open the cluster store from kv. It's ok if we don't find it.
	state, err := tryLoadPersistedState(ctx, cfg)
	if err != nil && !errors.Is(err, kv.NotFound) {
		return nil, err
	}
	c.Store.SetState(ctx, state)

	c.gossip, err = gossip.New(c.Gossip)
	if err != nil {
		return nil, err
	}

	c.R.Prod("cluster", c)
	c.L.Info("beginning cluster startup", c.Report().ZapFields()...)

	if !state.IsZero() {
		// If our store is valid, restart using the existing state.
		c.L.Info("existing cluster found in storage. restarting activities")
		host := c.Store.GetHost()
		host.Heartbeat = host.Heartbeat.Restart()
		c.SetNode(ctx, host)
		c.Pledge.ClusterKey = c.Key()
		if err := pledge_.Arbitrate(c.Pledge); err != nil {
			return nil, err
		}
	} else if len(c.Pledge.Peers) > 0 {
		// If our store is empty or invalid and peers were provided, attempt to join
		// the cluster.
		c.L.Info("no cluster found in storage. pledging to cluster instead")
		pledgeRes, err := pledge_.Pledge(ctx, c.Pledge)
		if err != nil {
			return nil, err
		}
		c.SetHost(ctx, node.Node{Key: pledgeRes.Key, Address: c.HostAddress})
		c.SetClusterKey(ctx, pledgeRes.ClusterKey)
		// gossip initial state manually through peers in order to build an
		// initial view of the cluster.
		c.L.Info("gossiping initial state through peer addresses")
		if err = c.gossipInitialState(ctx); err != nil {
			return c, err
		}
	} else {
		// If our store isn't valid, and we haven't received peers, assume we're
		// bootstrapping a new cluster.
		c.SetHost(ctx, node.Node{Key: 1, Address: c.HostAddress})
		c.SetClusterKey(ctx, uuid.New())
		c.L.Info("no peers provided, bootstrapping new cluster")
		c.Pledge.ClusterKey = c.Key()
		if err := pledge_.Arbitrate(c.Pledge); err != nil {
			return c, err
		}
	}

	// After we've successfully pledged, we can start gossiping cluster state.
	c.gossip.GoGossip(sCtx)

	// Periodically persist the cluster state.
	c.goFlushStore(sCtx)

	return c, nil
}

type cluster struct {
	Config
	store.Store
	gossip   *gossip.Gossip
	shutdown io.Closer
}

// Key implements the Cluster interface.
func (c *cluster) Key() uuid.UUID {
	return c.Store.PeekState().ClusterKey
}

// Host implements the Cluster interface.
func (c *cluster) Host() node.Node { return c.Store.GetHost() }

// HostKey implements the Cluster interface.
func (c *cluster) HostKey() node.Key { return c.Store.PeekState().HostKey }

// Nodes implements the Cluster interface.
func (c *cluster) Nodes() node.Group { return c.Store.PeekState().Nodes }

// Node implements the Cluster interface.
func (c *cluster) Node(key node.Key) (node.Node, error) {
	n, ok := c.Store.GetNode(key)
	if !ok {
		return n, NodeNotFound
	}
	return n, nil
}

// Resolve implements the Cluster interface.
func (c *cluster) Resolve(key node.Key) (address.Address, error) {
	n, err := c.Node(key)
	return n.Address, err
}

func (c *cluster) Close() error {
	return c.shutdown.Close()
}

func (c *cluster) gossipInitialState(ctx context.Context) error {
	nextAddr := iter.Endlessly(c.Pledge.Peers)
	for peerAddr := nextAddr(); peerAddr != ""; peerAddr = nextAddr() {
		if err := c.gossip.GossipOnceWith(ctx, peerAddr); err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			c.L.Error(
				"failed to gossip with peer",
				zap.String("peer", string(peerAddr)),
				zap.Error(err),
			)
		}
		if len(c.Store.CopyState().Nodes) > 1 {
			break
		}
	}
	return nil
}

func (c *cluster) goFlushStore(ctx signal.Context) {
	if c.Storage != nil {
		flush := &observe.FlushSubscriber[State]{
			Key:         c.StorageKey,
			MinInterval: c.StorageFlushInterval,
			Store:       c.Storage,
			Encoder:     c.EncoderDecoder,
		}
		flush.FlushSync(ctx, c.Store.CopyState())
		c.OnChange(func(ctx context.Context, state State) { flush.Flush(ctx, state) })
		ctx.Go(func(ctx context.Context) error {
			<-ctx.Done()
			flush.FlushSync(ctx, c.Store.CopyState())
			return ctx.Err()
		}, signal.WithKey("flush"))
	}
}

func tryLoadPersistedState(ctx context.Context, cfg Config) (store.State, error) {
	var state store.State
	if cfg.Storage == nil {
		return state, nil
	}
	encoded, err := cfg.Storage.Get(ctx, cfg.StorageKey)
	if err != nil {
		return state, lo.Ternary(errors.Is(err, kv.NotFound), nil, err)
	}
	return state, cfg.EncoderDecoder.Decode(ctx, encoded, &state)
}

func newConfig(ctx context.Context, configs []Config) (Config, error) {
	cfg, err := config.New(DefaultConfig, configs...)
	if err != nil {
		return Config{}, err
	}
	store_ := store.New(ctx)
	cfg.Gossip.Store = store_
	cfg.Pledge.Candidates = func() node.Group { return store_.CopyState().Nodes }
	cfg.Gossip.Instrumentation = cfg.Instrumentation.Sub("gossip")
	cfg.Pledge.Instrumentation = cfg.Instrumentation.Sub("pledge")
	return cfg, nil
}
