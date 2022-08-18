// Package cluster provides an interface for joining a cluster of nodes and
// exchanging state through an SI gossip model. nodes can join the cluster without
// needing to know all members. Cluster will automatically manage the membership of
// new nodes by assigning them unique IDs and keeping them in sync with their peers.
// To Join a cluster, simply use cluster.Join.
package cluster

import (
	"context"
	"github.com/arya-analytics/aspen/internal/cluster/gossip"
	pledge_ "github.com/arya-analytics/aspen/internal/cluster/pledge"
	"github.com/arya-analytics/aspen/internal/cluster/store"
	"github.com/arya-analytics/aspen/internal/node"
	"github.com/arya-analytics/x/address"
	"github.com/arya-analytics/x/alamos"
	"github.com/arya-analytics/x/iter"
	"github.com/arya-analytics/x/kv"
	"github.com/arya-analytics/x/observe"
	"github.com/arya-analytics/x/signal"
	"github.com/cockroachdb/errors"
	"go.uber.org/zap"
)

// State represents the current state of the cluster as seen from the host node.
type State = store.State

// ErrNotFound is returned when a node cannot be found in the cluster.
var ErrNotFound = errors.New("[cluster] - node not found")

// Cluster represents a group of nodes that can exchange their state with each other.
type Cluster interface {
	HostResolver
	// Nodes returns a node.Group of all nodes in the cluster. The returned map is not safe to modify. To modify,
	// use node.Group.Copy().
	Nodes() node.Group
	// Node returns the member Node with the given ID.
	Node(id node.ID) (node.Node, error)
	// Config returns the configuration parameters used by the cluster.
	Config() Config
	// Observable returns can be used to monitor changes to the cluster state. Be careful not to modify the
	// contents of the returned State.
	observe.Observable[State]
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
func Join(ctx signal.Context, addr address.Address, peers []address.Address, cfg Config) (Cluster, error) {
	cfg = cfg.Merge(DefaultConfig())
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	alamos.AttachReporter(cfg.Experiment, "cluster", alamos.Debug, cfg)
	cfg.Logger.Infow("beginning cluster startup", "storageFlushInterval", cfg.StorageFlushInterval)

	// Attempt to open the cluster store from kv. It's ok if we don't find it.
	s, err := openStore(cfg)
	if err != nil && err != kv.NotFound {
		return nil, err
	}

	c := &cluster{Store: s, cfg: cfg}

	// We need to open the gossip service before we can join the cluster in
	// order to prevent issues with incoming requests receiving a nil transport
	// handler.
	g, err := gossip.New(s, c.cfg.Gossip)
	if err != nil {
		return nil, err
	}

	// If our store is empty or invalid and peers were provided, attempt to join
	// the cluster.
	if !s.Valid() && len(peers) != 0 {
		cfg.Logger.Infow(
			"no existing cluster found in storage. pledging to cluster instead",
		)
		id, err := pledge(ctx, peers, c)
		if err != nil {
			return nil, err
		}
		c.Store.SetHost(node.Node{ID: id, Address: addr})
		// operationSender initial cluster state, so we can contact it for
		// information on other nodes instead of peers.
		cfg.Logger.Info("gossiping initial state through peer addresses")
		if err = gossipInitialState(ctx, c.Store, c.cfg, peers, g); err != nil {
			return c, err
		}
	} else if !s.Valid() && len(peers) == 0 {
		// If our store isn't valid, and we haven't received peers, assume we're
		// bootstrapping a new cluster.
		c.Store.SetHost(node.Node{ID: 1, Address: addr})
		cfg.Logger.Infow(
			"no peers provided, bootstrapping new cluster",
		)
		if err := pledge_.Arbitrate(c.Nodes, c.cfg.Pledge); err != nil {
			return c, err
		}
	} else {

		// If our store is valid, restart using the existing state.
		cfg.Logger.Infow(
			"existing cluster found in storage. restarting activities",
		)

		host := c.Store.GetHost()
		host.Heartbeat = host.Heartbeat.Restart()
		c.Store.Set(host)

		if err := pledge_.Arbitrate(c.Nodes, c.cfg.Pledge); err != nil {
			return nil, err
		}
	}

	// After we've successfully pledged, we can start gossiping cluster state.
	g.GoGossip(ctx)

	// Periodically persist the cluster state.
	goFlushStore(ctx, cfg, s)

	return c, nil
}

type cluster struct {
	cfg Config
	store.Store
}

// Host implements the Cluster interface.
func (c *cluster) Host() node.Node { return c.Store.GetHost() }

// HostID implements the Cluster interface.
func (c *cluster) HostID() node.ID { return c.Store.ReadState().HostID }

// Nodes implements the Cluster interface.
func (c *cluster) Nodes() node.Group { return c.Store.ReadState().Nodes }

// Node implements the Cluster interface.
func (c *cluster) Node(id node.ID) (node.Node, error) {
	n, ok := c.Store.Get(id)
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

// Config implements the Cluster interface.
func (c *cluster) Config() Config { return c.cfg }

func openStore(cfg Config) (store.Store, error) {
	s := store.New()
	if cfg.Storage == nil {
		return s, nil
	}
	return s, kv.Load(cfg.Storage, cfg.StorageKey, s)
}

func pledge(ctx context.Context, peers []address.Address, c *cluster) (node.ID, error) {
	candidates := func() node.Group { return c.Store.CopyState().Nodes }
	return pledge_.Pledge(ctx, peers, candidates, c.cfg.Pledge)
}

func gossipInitialState(
	ctx context.Context,
	s store.Store,
	cfg Config,
	peers []address.Address,
	g *gossip.Gossip,
) error {
	nextAddr := iter.Endlessly(peers)
	for peerAddr := nextAddr(); peerAddr != ""; peerAddr = nextAddr() {
		if err := g.GossipOnceWith(ctx, peerAddr); err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			cfg.Logger.Error("failed to gossip with peer",
				zap.String("peer", string(peerAddr)),
				zap.Error(err),
			)
		}
		if len(s.CopyState().Nodes) > 1 {
			break
		}
	}
	return nil
}

func goFlushStore(ctx signal.Context, cfg Config, s store.Store) {
	if cfg.Storage != nil {
		flush := &observe.FlushSubscriber[State]{
			Key:         cfg.StorageKey,
			MinInterval: cfg.StorageFlushInterval,
			Store:       cfg.Storage,
			Logger:      cfg.Logger,
		}
		flush.FlushSync(s.CopyState())
		s.OnChange(flush.Flush)
		ctx.Go(func(ctx context.Context) error {
			<-ctx.Done()
			flush.FlushSync(s.CopyState())
			return ctx.Err()
		})
	}
}
