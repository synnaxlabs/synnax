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
// To Join a cluster, simply use cluster.Join.
package cluster

import (
	"context"
	"github.com/cockroachdb/errors"
	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
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
	// Node returns the member Node with the given ID.
	Node(id node.ID) (node.Node, error)
	// Observable returns can be used to monitor changes to the cluster state. Be careful not to modify the
	// contents of the returned State.
	observe.Observable[State]
	// Reader allows reading the current state of the cluster.
	storex.Reader[State]
	io.Closer
}

// Resolver is used to resolve a reachable address for a node in the cluster.
type Resolver interface {
	// Resolve resolves the address of a node with the given ID.
	Resolve(id node.ID) (address.Address, error)
}

type Host interface {
	// Host returns the host Node (i.e. the node that Host is called on).
	Host() node.Node
	// HostID returns the ID of the host node.
	HostID() node.ID
}

type HostResolver interface {
	Resolver
	Host
}

// Join joins the host node to the cluster and begins gossiping its state. The
// node will spread addr as its listening address. A set of peer addresses
// (other nodes in the cluster) must be provided when joining an existing cluster
// for the first time. If restarting a node that is already a member of a cluster,
// the peer addresses can be left empty; Join will attempt to load the existing
// cluster state from storage (see Config.Storage and Config.StorageKey).
// If provisioning a new cluster, ensure that all storage for previous clusters
// is removed and provide no peers.
func Join(ctx context.Context, cfgs ...Config) (Cluster, error) {
	cfg, err := config.OverrideAndValidate(DefaultConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	sCtx, cancel := signal.WithCancel(ctx)
	store_ := store.New()
	cfg.Gossip.Store = store_
	cfg.Pledge.Candidates = func() node.Group { return store_.CopyState().Nodes }

	// Attempt to open the cluster store from kv. It's ok if we don't find it.
	state, err := tryLoadPersistedState(ctx, cfg)
	if err != nil && !errors.Is(err, kv.NotFound) {
		return nil, err
	}
	store_.SetState(state)

	c := &cluster{Store: store_, shutdown: cancel, wg: sCtx, cfg: cfg}
	if err != nil {
		return nil, err
	}
	c.gossip, err = gossip.New(c.cfg.Gossip)

	alamos.AttachReporter(ctx, "cluster", alamos.Debug, c.cfg)
	log := alamos.L(ctx)
	log.Info("beginning cluster startup", c.cfg.Report().LogArgs()...)

	// If our store is empty or invalid and peers were provided, attempt to join
	// the cluster.
	if !state.IsZero() {
		// If our store is valid, restart using the existing state.
		log.Info("existing cluster found in storage. restarting activities")

		host := c.Store.GetHost()
		host.Heartbeat = host.Heartbeat.Restart()
		c.Store.SetNode(host)
		c.cfg.Pledge.ClusterKey = c.Key()
		if err := pledge_.Arbitrate(ctx, c.cfg.Pledge); err != nil {
			return nil, err
		}
	} else if len(c.cfg.Pledge.Peers) != 0 {
		log.Info("no cluster found in storage. pledging to cluster instead")
		pledgeRes, err := pledge_.Pledge(ctx, c.cfg.Pledge)
		if err != nil {
			return nil, err
		}
		c.Store.SetHost(node.Node{ID: pledgeRes.ID, Address: c.cfg.HostAddress})
		c.Store.SetClusterKey(pledgeRes.ClusterKey)
		// operationSender initial cluster state, so we can contact it for
		// information on other nodes instead of peers.

		log.Info("gossiping initial state through peer addresses")
		if err = c.gossipInitialState(ctx); err != nil {
			return c, err
		}
	} else {
		// If our store isn't valid, and we haven't received peers, assume we're
		// bootstrapping a new cluster.
		c.Store.SetHost(node.Node{ID: 1, Address: c.cfg.HostAddress})
		c.SetClusterKey(uuid.New())
		log.Info("no peers provided, bootstrapping new cluster")
		c.cfg.Pledge.ClusterKey = c.Key()
		if err := pledge_.Arbitrate(ctx, c.cfg.Pledge); err != nil {
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
	store.Store
	cfg      Config
	gossip   *gossip.Gossip
	shutdown context.CancelFunc
	wg       signal.WaitGroup
}

// Key implements the Cluster interface.
func (c *cluster) Key() uuid.UUID {
	return c.Store.PeekState().ClusterKey
}

// Host implements the Cluster interface.
func (c *cluster) Host() node.Node { return c.Store.GetHost() }

// HostID implements the Cluster interface.
func (c *cluster) HostID() node.ID { return c.Store.PeekState().HostID }

// Nodes implements the Cluster interface.
func (c *cluster) Nodes() node.Group { return c.Store.PeekState().Nodes }

// Node implements the Cluster interface.
func (c *cluster) Node(id node.ID) (node.Node, error) {
	n, ok := c.Store.GetNode(id)
	if !ok {
		return n, NodeNotFound
	}
	return n, nil
}

// Resolve implements the Cluster interface.
func (c *cluster) Resolve(id node.ID) (address.Address, error) {
	n, err := c.Node(id)
	return n.Address, err
}

func (c *cluster) Close() error {
	c.shutdown()
	return c.wg.Wait()
}

func tryLoadPersistedState(ctx context.Context, cfg Config) (store.State, error) {
	var state store.State
	encoded, err := cfg.Storage.Get(ctx, cfg.StorageKey)
	if err != nil {
		return state, lo.Ternary(errors.Is(err, kv.NotFound), nil, err)
	}
	return state, cfg.EncoderDecoder.Decode(encoded, &state)
}

func (c *cluster) gossipInitialState(ctx context.Context) error {
	nextAddr := iter.Endlessly(c.cfg.Pledge.Peers)
	for peerAddr := nextAddr(); peerAddr != ""; peerAddr = nextAddr() {
		if err := c.gossip.GossipOnceWith(ctx, peerAddr); err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			alamos.L(ctx).Error(
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
	if c.cfg.Storage != nil {
		flush := &observe.FlushSubscriber[State]{
			Key:         c.cfg.StorageKey,
			MinInterval: c.cfg.StorageFlushInterval,
			Store:       c.cfg.Storage,
			Encoder:     c.cfg.EncoderDecoder,
		}
		flush.FlushSync(ctx, c.Store.CopyState())
		c.OnChange(func(state State) { flush.Flush(ctx, state) })
		ctx.Go(func(ctx context.Context) error {
			<-ctx.Done()
			flush.FlushSync(ctx, c.Store.CopyState())
			return ctx.Err()
		}, signal.WithKey("flush"))
	}
}
