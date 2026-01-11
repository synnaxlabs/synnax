// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package cluster provides an interface for joining a Cluster of nodes and
// exchanging state through an SI gossip model. nodes can join the Cluster without
// needing to know all members. Cluster will automatically manage the membership of
// new nodes by assigning them unique keys and keeping them in sync with their peers.
// To Open a Cluster, simply use cluster.Open.
package cluster

import (
	"context"
	"io"
	"time"

	"github.com/samber/lo"
	"github.com/synnaxlabs/aspen/internal/cluster/gossip"
	pledge_ "github.com/synnaxlabs/aspen/internal/cluster/pledge"
	"github.com/synnaxlabs/aspen/internal/cluster/store"
	"github.com/synnaxlabs/aspen/node"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/signal"
	xslices "github.com/synnaxlabs/x/slices"
	"github.com/synnaxlabs/x/uuid"
	"go.uber.org/zap"
)

// State represents the current state of the Cluster as seen from the host node.
type State = store.State

// Change is information about a change to the Cluster's state.
type Change = store.Change

// NodeNotFound is returned when a node cannot be found in the Cluster.
var NodeNotFound = errors.Wrap(query.NotFound, "node not found")

func nodeNotFoundErr(key node.Key) error {
	return errors.Wrapf(NodeNotFound, "node %d", key)
}

// Open joins the host node to the Cluster and begins gossiping its state. The
// node will spread addr as its listening address. A set of peer addresses
// (other nodes in the Cluster) must be provided when joining an existing Cluster
// for the first time. If restarting a node that is already a member of a Cluster,
// the peer addresses can be left empty; Open will attempt to load the existing
// Cluster state from storage (see Config.Storage and Config.StorageKey).
// If provisioning a new Cluster, ensure that all storage for previous clusters
// is removed and provide no peers.
func Open(ctx context.Context, configs ...Config) (*Cluster, error) {
	cfg, err := newConfig(ctx, configs)
	if err != nil {
		return nil, err
	}

	sCtx, cancel := signal.WithCancel(cfg.T.Transfer(ctx, context.Background()))

	c := &Cluster{
		Store:    cfg.Gossip.Store,
		shutdown: signal.NewHardShutdown(sCtx, cancel),
		Config:   cfg,
	}

	// Attempt to open the Cluster store from kv. It's ok if we don't find it.
	state, err := tryLoadPersistedState(ctx, cfg)
	if err != nil && !errors.Is(err, kv.NotFound) {
		return nil, err
	}
	c.SetState(ctx, state)

	c.gossip, err = gossip.New(c.Gossip)
	if err != nil {
		return nil, err
	}

	c.R.Prod("cluster", c)
	c.L.Info("beginning cluster startup")
	c.L.Debug("configuration", cfg.Report().ZapFields()...)

	if !state.IsZero() {
		// If our store is valid, restart using the existing state.
		c.L.Info("existing cluster found in storage. restarting activities")
		host := c.GetHost()
		host.Heartbeat = host.Heartbeat.Restart()
		c.SetNode(ctx, host)
		c.Pledge.ClusterKey = c.Key()
		if err := pledge_.Arbitrate(c.Pledge); err != nil {
			return nil, err
		}
	} else if len(c.Pledge.Peers) > 0 {
		// If our store is empty or invalid and peers were provided, attempt to join
		// the Cluster.
		c.L.Info("no cluster found in storage. pledging to Cluster instead")
		pledgeRes, err := pledge_.Pledge(ctx, c.Pledge)
		if err != nil {
			return nil, err
		}
		c.SetHost(ctx, node.Node{Key: pledgeRes.Key, Address: c.HostAddress})
		c.SetClusterKey(ctx, pledgeRes.ClusterKey)
		// gossip initial state manually through peers in order to build an
		// initial view of the Cluster.
		c.L.Info("gossiping initial state through peer addresses")
		if err = c.gossipInitialState(ctx); err != nil {
			return c, err
		}
	} else {
		// If our store isn't valid, and we haven't received peers, assume we're
		// bootstrapping a new Cluster.
		c.SetHost(ctx, node.Node{Key: 1, Address: c.HostAddress})
		clusterKey := uuid.New()
		c.SetClusterKey(ctx, clusterKey)
		c.L.Info("no peers provided, bootstrapping new cluster", zap.Stringer("cluster_key", clusterKey))
		c.Pledge.ClusterKey = c.Key()
		if err = pledge_.Arbitrate(c.Pledge); err != nil {
			return c, err
		}
	}

	// After we've successfully pledged, we can start gossiping Cluster state.
	c.gossip.GoGossip(sCtx)

	// Periodically persist the Cluster state.
	c.goFlushStore(sCtx)

	return c, nil
}

type Cluster struct {
	Config
	store.Store
	gossip   *gossip.Gossip
	shutdown io.Closer
}

// Key implements the Cluster interface.
func (c *Cluster) Key() uuid.UUID {
	s, release := c.PeekState()
	defer release()
	return s.ClusterKey
}

// Host implements the Cluster interface.
func (c *Cluster) Host() node.Node {
	return c.GetHost()
}

// HostKey implements the Cluster interface.
func (c *Cluster) HostKey() node.Key {
	s, release := c.PeekState()
	defer release()
	return s.HostKey
}

// Nodes implements the Cluster interface.
func (c *Cluster) Nodes() node.Group {
	s, release := c.PeekState()
	defer release()
	return s.Nodes
}

// Node implements the Cluster interface.
func (c *Cluster) Node(key node.Key) (node.Node, error) {
	n, ok := c.GetNode(key)
	if !ok {
		return n, nodeNotFoundErr(key)
	}
	return n, nil
}

// Resolve implements the Cluster interface.
func (c *Cluster) Resolve(key node.Key) (address.Address, error) {
	n, err := c.Node(key)
	return n.Address, err
}

func (c *Cluster) Close() error { return c.shutdown.Close() }

func (c *Cluster) gossipInitialState(ctx context.Context) error {
	for peer := range xslices.IterEndlessly(c.Pledge.Peers) {
		if err := c.gossip.GossipOnceWith(ctx, peer); err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			c.L.Error(
				"failed to gossip with peer",
				zap.String("peer", string(peer)),
				zap.Error(err),
			)
		}
		if len(c.Store.CopyState().Nodes) > 1 {
			break
		}
	}
	return nil
}

func (c *Cluster) goFlushStore(sCtx signal.Context) {
	if c.Storage != nil {
		flush := &kv.Subscriber[State]{
			Key:         c.StorageKey,
			MinInterval: c.StorageFlushInterval,
			Store:       c.Storage,
			Encoder:     c.Codec,
		}
		flush.FlushSync(sCtx, c.CopyState())
		c.OnChange(func(_ context.Context, change Change) {
			select {
			case <-sCtx.Done():
				return
			default:
				flush.Flush(sCtx, change.State)
			}
		})
		sCtx.Go(func(ctx context.Context) error {
			<-ctx.Done()
			flush.FlushSync(ctx, c.CopyState())
			return ctx.Err()
		},
			signal.WithKey("flush"),
			signal.WithRetryOnPanic(),
			signal.WithRetryScale(1.05),
			signal.WithBaseRetryInterval(1*time.Second),
		)
	}
}

func tryLoadPersistedState(ctx context.Context, cfg Config) (store.State, error) {
	var state store.State
	if cfg.Storage == nil {
		return state, nil
	}
	encoded, closer, err := cfg.Storage.Get(ctx, cfg.StorageKey)
	if err != nil {
		return state, lo.Ternary(errors.Is(err, kv.NotFound), nil, err)
	}
	err = cfg.Codec.Decode(ctx, encoded, &state)
	err = errors.Combine(err, closer.Close())
	return state, err
}

func newConfig(ctx context.Context, configs []Config) (Config, error) {
	cfg, err := config.New(DefaultConfig, configs...)
	if err != nil {
		return Config{}, err
	}
	store_ := store.New(ctx)
	cfg.Gossip.Store = store_
	cfg.Pledge.Candidates = func() node.Group { return store_.CopyState().Nodes }
	cfg.Gossip.Instrumentation = cfg.Child("gossip")
	cfg.Pledge.Instrumentation = cfg.Child("pledge")
	return cfg, nil
}
