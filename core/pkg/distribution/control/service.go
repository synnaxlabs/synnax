// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package control

import (
	"context"
	"io"
	"sync"
	"time"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/distribution/proxy"
	"github.com/synnaxlabs/synnax/pkg/storage/ts"
	"github.com/synnaxlabs/x/breaker"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

// ServiceConfig configures the distribution-layer control service.
type ServiceConfig struct {
	// Instrumentation is for logging, tracing, and metrics.
	//
	// [OPTIONAL] - Defaults to noop instrumentation.
	alamos.Instrumentation
	// Cluster is the cluster-membership view, used to discover peers and to open and
	// close their subscriptions as the topology changes.
	//
	// [REQUIRED]
	Cluster cluster.Cluster
	// TS is the storage-layer time-series database that arbitrates control over the
	// channels leased to the host node.
	//
	// [REQUIRED]
	TS *ts.DB
	// Transport routes control retrieves and subscriptions between Cores.
	//
	// [REQUIRED]
	Transport Transport
	// Breaker configures the retry behavior of peer subscriptions.
	//
	// [OPTIONAL] - Defaults to DefaultServiceConfig.Breaker.
	Breaker breaker.Config
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// DefaultServiceConfig is the default configuration for the control service. A peer
// subscription retries until the peer leaves the cluster, starting fast because a
// freshly joined node usually only needs another moment to bind its handler.
var DefaultServiceConfig = ServiceConfig{
	Breaker: breaker.Config{
		BaseInterval: 50 * time.Millisecond,
		Scale:        1.1,
		MaxRetries:   breaker.InfiniteRetries,
	},
}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("distribution.control")
	v.NotNil("cluster", c.Cluster)
	v.NotNil("ts", c.TS)
	v.NotNil("transport", c.Transport)
	return v.Error()
}

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.Cluster = override.Nil(c.Cluster, other.Cluster)
	c.TS = override.Nil(c.TS, other.TS)
	c.Transport = override.Nil(c.Transport, other.Transport)
	c.Breaker = c.Breaker.Override(other.Breaker)
	return c
}

// peer is the host's view of one remote Core's control state, held so incoming
// snapshots can be turned back into the transfers that produced them.
type peer struct {
	states map[channel.Key]State
	stop   context.CancelFunc
}

// Service answers control-state reads and subscriptions for the whole cluster. Control
// is arbitrated per channel by its leaseholder, so the service reads the host's state
// from storage, routes the rest to their leaseholders, and merges every peer's changes
// into one stream of transfers.
//
// The Service must be closed after use.
type Service struct {
	cfg      ServiceConfig
	router   proxy.BatchFactory[channel.Key]
	updates  observe.Observer[Update]
	shutdown io.Closer
	// disconnectStorage stops the host's storage subscription.
	disconnectStorage observe.Disconnect
	// disconnectCluster stops the membership subscription.
	disconnectCluster observe.Disconnect
	mu                struct {
		sync.Mutex
		// peers holds one entry per remote Core the host currently subscribes to.
		peers map[node.Key]*peer
		// closed stops a sync racing Close from opening another subscription.
		closed bool
	}
}

// OpenService opens the distribution-layer control service using the provided
// configuration(s). Fields defined in each subsequent configuration override those in
// previous configurations.
//
// If the returned error is nil, the Service must be closed after use.
func OpenService(ctx context.Context, cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(DefaultServiceConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	s := &Service{
		cfg:     cfg,
		router:  proxy.BatchFactory[channel.Key](cfg.Cluster.HostKey()),
		updates: observe.New[Update](),
	}
	s.mu.peers = make(map[node.Key]*peer)
	sCtx, cancel := signal.WithCancel(
		context.WithoutCancel(ctx),
		signal.WithInstrumentation(cfg.Instrumentation),
	)
	s.shutdown = signal.NewHardShutdown(sCtx, cancel)
	cfg.Transport.RetrieveServer().BindHandler(s.handleRetrieve)
	cfg.Transport.SubscribeServer().BindHandler(
		func(ctx context.Context, stream SubscribeStream) error {
			return s.handleSubscribe(ctx, sCtx.Done(), stream)
		},
	)
	s.disconnectStorage = cfg.TS.OnControlUpdate(
		func(ctx context.Context, u ts.ControlUpdate) {
			s.updates.Notify(ctx, updateFromStorage(u))
		},
	)
	s.disconnectCluster = cfg.Cluster.OnChange(
		func(ctx context.Context, _ cluster.Change) { s.syncPeers(ctx, sCtx) },
	)
	s.syncPeers(ctx, sCtx)
	return s, nil
}

// Retrieve returns the control state of the given channels from across the cluster.
// Passing no keys returns the state of every controlled channel on every node. Channels
// that no subject controls are absent from the result.
func (s *Service) Retrieve(
	ctx context.Context,
	keys ...channel.Key,
) ([]State, error) {
	if len(keys) == 0 {
		return s.retrieveAll(ctx)
	}
	batch := s.router.Batch(keys)
	states := statesFromStorage(s.cfg.TS.ControlStates())
	out := filterByKeys(states, batch.Gateway)
	for target, targetKeys := range batch.Peers {
		res, err := s.retrievePeer(ctx, target, targetKeys)
		if err != nil {
			return nil, err
		}
		out = append(out, res...)
	}
	return out, nil
}

// OnChange registers handler to receive every control transfer that occurs anywhere in
// the cluster, whether the host arbitrated it or a peer did. The returned function
// unregisters the handler.
func (s *Service) OnChange(
	handler func(context.Context, Update),
) observe.Disconnect {
	return s.updates.OnChange(handler)
}

// Close stops every peer subscription and releases the service's resources.
func (s *Service) Close() error {
	s.disconnectCluster()
	s.disconnectStorage()
	s.mu.Lock()
	s.mu.closed = true
	for _, p := range s.mu.peers {
		p.stop()
	}
	clear(s.mu.peers)
	s.mu.Unlock()
	return s.shutdown.Close()
}

func (s *Service) retrieveAll(ctx context.Context) ([]State, error) {
	out := statesFromStorage(s.cfg.TS.ControlStates())
	host := s.cfg.Cluster.HostKey()
	for _, n := range s.cfg.Cluster.Nodes() {
		if n.Key == host {
			continue
		}
		res, err := s.retrievePeer(ctx, n.Key, nil)
		if err != nil {
			return nil, err
		}
		out = append(out, res...)
	}
	return out, nil
}

func (s *Service) retrievePeer(
	ctx context.Context,
	target node.Key,
	keys channel.Keys,
) ([]State, error) {
	addr, err := s.cfg.Cluster.Resolve(target)
	if err != nil {
		return nil, err
	}
	res, err := s.cfg.Transport.RetrieveClient().
		Send(ctx, addr, RetrieveRequest{Keys: keys})
	if err != nil {
		return nil, errors.Wrapf(
			err,
			"failed to retrieve control state from %v",
			target,
		)
	}
	return res.States, nil
}

func (s *Service) handleRetrieve(
	_ context.Context,
	req RetrieveRequest,
) (RetrieveResponse, error) {
	states := statesFromStorage(s.cfg.TS.ControlStates())
	if len(req.Keys) > 0 {
		states = filterByKeys(states, req.Keys)
	}
	return RetrieveResponse{States: states}, nil
}

// handleSubscribe serves a peer's subscription, sending the host's complete control
// state on open and after every transfer the host arbitrates. Each message supersedes
// the last, so a subscriber that falls behind converges rather than diverging, and the
// send is dropped instead of stalling the storage engine.
func (s *Service) handleSubscribe(
	ctx context.Context,
	stop <-chan struct{},
	stream SubscribeStream,
) error {
	changes := make(chan struct{}, 1)
	disconnect := s.cfg.TS.OnControlUpdate(func(context.Context, ts.ControlUpdate) {
		select {
		case changes <- struct{}{}:
		default:
		}
	})
	defer disconnect()
	send := func() error {
		return stream.Send(SubscribeResponse{
			States: statesFromStorage(s.cfg.TS.ControlStates()),
		})
	}
	if err := send(); err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-stop:
			return nil
		case <-changes:
			if err := send(); err != nil {
				return err
			}
		}
	}
}

// syncPeers opens a subscription to every peer the host does not already subscribe to.
// Nodes are never removed from the cluster's view, so a subscription is only ever
// opened here, never closed.
func (s *Service) syncPeers(ctx context.Context, sCtx signal.Context) {
	host := s.cfg.Cluster.HostKey()
	for _, n := range s.cfg.Cluster.Nodes() {
		if n.Key != host {
			s.addPeer(ctx, sCtx, n.Key)
		}
	}
}

// addPeer opens a subscription to target and starts reading it, doing nothing if the
// host already subscribes to target or the service is closed. The stream is opened here
// rather than inside the reader so that the subscription is live before addPeer
// returns, leaving no window in which a joining peer's transfers go unseen.
func (s *Service) addPeer(ctx context.Context, sCtx signal.Context, target node.Key) {
	peerCtx, stop := context.WithCancel(context.WithoutCancel(ctx))
	s.mu.Lock()
	_, subscribed := s.mu.peers[target]
	if subscribed || s.mu.closed {
		s.mu.Unlock()
		stop()
		return
	}
	// The entry is claimed before the stream opens so that a concurrent sync cannot
	// open a second subscription to the same peer.
	s.mu.peers[target] = &peer{states: make(map[channel.Key]State), stop: stop}
	s.mu.Unlock()
	stream, err := s.openPeerStream(peerCtx, target)
	sCtx.Go(
		func(context.Context) error {
			return s.subscribeToPeer(peerCtx, target, stream, err)
		},
		signal.WithKey("control_peer_"+target.String()),
	)
}

// subscribeToPeer reads stream until it fails, then reopens it under backoff, until the
// peer leaves the cluster or the service closes. stream and openErr carry the result of
// the caller's first open attempt.
func (s *Service) subscribeToPeer(
	ctx context.Context,
	target node.Key,
	stream SubscribeClientStream,
	openErr error,
) error {
	b, err := breaker.NewBreaker(ctx, s.cfg.Breaker)
	if err != nil {
		return err
	}
	for {
		if openErr == nil {
			b.Reset()
			openErr = s.readPeerStream(ctx, target, stream)
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		s.cfg.L.Debug(
			"control subscription to peer ended, retrying",
			zap.Stringer("node", target),
			zap.Error(openErr),
		)
		if !b.Wait() {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			return errors.Wrapf(
				openErr,
				"gave up subscribing to control state on %v",
				target,
			)
		}
		stream, openErr = s.openPeerStream(ctx, target)
	}
}

func (s *Service) openPeerStream(
	ctx context.Context,
	target node.Key,
) (SubscribeClientStream, error) {
	addr, err := s.cfg.Cluster.Resolve(target)
	if err != nil {
		return nil, err
	}
	return s.cfg.Transport.SubscribeClient().Stream(ctx, addr)
}

func (s *Service) readPeerStream(
	ctx context.Context,
	target node.Key,
	stream SubscribeClientStream,
) error {
	defer func() {
		if closeErr := stream.CloseSend(); closeErr != nil {
			s.cfg.L.Debug(
				"failed to close control subscription",
				zap.Stringer("node", target),
				zap.Error(closeErr),
			)
		}
	}()
	for {
		res, err := stream.Receive()
		if err != nil {
			return err
		}
		s.applyPeerStates(ctx, target, res.States)
	}
}

// applyPeerStates replaces the host's view of target's control state and notifies
// subscribers of the transfers that carry the previous view to the new one.
func (s *Service) applyPeerStates(
	ctx context.Context,
	target node.Key,
	states []State,
) {
	next := statesByChannel(states)
	s.mu.Lock()
	p, ok := s.mu.peers[target]
	if !ok {
		s.mu.Unlock()
		return
	}
	transfers := diff(p.states, next)
	p.states = next
	s.mu.Unlock()
	if len(transfers) > 0 {
		s.updates.Notify(ctx, Update{Transfers: transfers})
	}
}

func filterByKeys(states []State, keys channel.Keys) []State {
	if len(keys) == 0 {
		return nil
	}
	requested := set.New(keys...)
	out := make([]State, 0, len(keys))
	for _, s := range states {
		if requested.Contains(s.Resource) {
			out = append(out, s)
		}
	}
	return out
}
