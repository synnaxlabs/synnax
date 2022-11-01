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
	"github.com/synnaxlabs/aspen/internal/cluster/gossip"
	pledge_ "github.com/synnaxlabs/aspen/internal/cluster/pledge"
	"github.com/synnaxlabs/aspen/internal/cluster/store"
	"github.com/synnaxlabs/aspen/internal/node"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/alamos"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/iter"
	"github.com/synnaxlabs/x/kv"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/signal"
	storex "github.com/synnaxlabs/x/store"
	"go.uber.org/zap"
)

// State represents the current state of the cluster as seen from the host node.
type State = store.State

// ErrNotFound is returned when a node cannot be found in the cluster.
var ErrNotFound = errors.New("[cluster] - node not found")

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
func Join(ctx signal.Context, cfgs ...Config) (Cluster, error) {
	c, err := newFromConfigs(cfgs...)
	if err != nil {
		return nil, err
	}

	// Attempt to open the cluster store from kv. It's ok if we don't find it.
	if err := c.loadStateFromStorage(); err != nil && !errors.Is(err, kv.NotFound) {
		return nil, err
	}

	alamos.AttachReporter(c.cfg.Experiment, "cluster", alamos.Debug, c.cfg)
	c.cfg.Logger.Infow("beginning cluster startup", c.cfg.Report().LogArgs()...)

	// If our store is empty or invalid and peers were provided, attempt to join
	// the cluster.
	if !c.Store.Valid() && len(c.cfg.Pledge.Peers) != 0 {
		c.cfg.Logger.Infow(
			"no existing cluster found in storage. pledging to cluster instead",
		)
		pledgeRes, err := pledge_.Pledge(ctx, c.cfg.Pledge)
		if err != nil {
			return nil, err
		}
		c.Store.SetHost(node.Node{ID: pledgeRes.ID, Address: c.cfg.HostAddress})
		c.Store.SetClusterKey(pledgeRes.ClusterKey)
		// operationSender initial cluster state, so we can contact it for
		// information on other nodes instead of peers.

		c.cfg.Logger.Info("gossiping initial state through peer addresses")
		if err = c.gossipInitialState(ctx); err != nil {
			return c, err
		}
	} else if !c.Store.Valid() && len(c.cfg.Pledge.Peers) == 0 {
		// If our store isn't valid, and we haven't received peers, assume we're
		// bootstrapping a new cluster.
		c.Store.SetHost(node.Node{ID: 1, Address: c.cfg.HostAddress})
		c.SetClusterKey(uuid.New())
		c.cfg.Logger.Infow(
			"no peers provided, bootstrapping new cluster",
		)
		c.cfg.Pledge.ClusterKey = c.Key()
		if err := pledge_.Arbitrate(c.cfg.Pledge); err != nil {
			return c, err
		}
	} else {

		// If our store is valid, restart using the existing state.
		c.cfg.Logger.Infow(
			"existing cluster found in storage. restarting activities",
		)

		host := c.Store.GetHost()
		host.Heartbeat = host.Heartbeat.Restart()
		c.Store.SetNode(host)
		c.cfg.Pledge.ClusterKey = c.Key()
		if err := pledge_.Arbitrate(c.cfg.Pledge); err != nil {
			return nil, err
		}
	}

	// After we've successfully pledged, we can start gossiping cluster state.
	c.gossip.GoGossip(ctx)

	// Periodically persist the cluster state.
	c.goFlushStore(ctx)

	return c, nil
}

type cluster struct {
	store.Store
	cfg    Config
	gossip *gossip.Gossip
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
		return n, ErrNotFound
	}
	return n, nil
}

// Resolve implements the Cluster interface.
func (c *cluster) Resolve(id node.ID) (address.Address, error) {
	n, err := c.Node(id)
	return n.Address, err
}

func (c *cluster) loadStateFromStorage() error {
	if c.cfg.Storage == nil {
		return nil
	}
	encoded, err := c.cfg.Storage.Get(c.cfg.StorageKey)
	if err != nil {
		return err
	}
	var state store.State
	if err := c.cfg.EncoderDecoder.Decode(encoded, &state); err != nil {
		return err
	}
	c.Store.SetState(state)
	return nil
}

func (c *cluster) gossipInitialState(ctx context.Context) error {
	nextAddr := iter.Endlessly(c.cfg.Pledge.Peers)
	for peerAddr := nextAddr(); peerAddr != ""; peerAddr = nextAddr() {
		if err := c.gossip.GossipOnceWith(ctx, peerAddr); err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			c.cfg.Logger.Error("failed to gossip with peer",
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
			Logger:      c.cfg.Logger,
			Encoder:     c.cfg.EncoderDecoder,
		}
		flush.FlushSync(c.Store.CopyState())
		c.OnChange(flush.Flush)
		ctx.Go(func(ctx context.Context) error {
			<-ctx.Done()
			flush.FlushSync(c.Store.CopyState())
			return ctx.Err()
		})
	}
}

func newFromConfigs(cfgs ...Config) (*cluster, error) {
	c := &cluster{Store: store.New()}
	base := DefaultConfig
	base.Gossip.Store = c.Store
	base.Pledge.Candidates = func() node.Group { return c.Store.CopyState().Nodes }
	var err error
	c.cfg, err = config.OverrideAndValidate(base, cfgs...)
	if err != nil {
		return nil, err
	}
	c.gossip, err = gossip.New(c.cfg.Gossip)
	return c, err
}
