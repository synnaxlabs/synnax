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

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/distribution/proxy"
	"github.com/synnaxlabs/synnax/pkg/storage/ts"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/observe"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/validate"
)

// ServiceConfig configures the distribution-layer control service.
type ServiceConfig struct {
	// Instrumentation is for logging, tracing, and metrics.
	//
	// [OPTIONAL] - Defaults to noop instrumentation.
	alamos.Instrumentation
	// HostResolver resolves the host node and the leaseholders of channels.
	//
	// [REQUIRED]
	HostResolver cluster.HostResolver
	// TS is the storage-layer time-series database that arbitrates control over the
	// channels leased to the host node.
	//
	// [REQUIRED]
	TS *ts.DB
	// Transport routes control retrieves and subscriptions between Cores.
	//
	// [REQUIRED]
	Transport Transport
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// Validate implements config.Config.
func (c ServiceConfig) Validate() error {
	v := validate.New("distribution.control")
	v.NotNil("host_resolver", c.HostResolver)
	v.NotNil("ts", c.TS)
	v.NotNil("transport", c.Transport)
	return v.Error()
}

// Override implements config.Config.
func (c ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.HostResolver = override.Nil(c.HostResolver, other.HostResolver)
	c.TS = override.Nil(c.TS, other.TS)
	c.Transport = override.Nil(c.Transport, other.Transport)
	return c
}

// Service answers control-state reads and subscriptions. Control is arbitrated per
// channel by its leaseholder, so the service reads the host's state from storage and
// routes the rest to their leaseholders.
//
// The Service must be closed after use.
type Service struct {
	cfg      ServiceConfig
	router   proxy.BatchFactory[channel.Key]
	updates  observe.Observer[Update]
	shutdown io.Closer
	// disconnectStorage stops the host's storage subscription.
	disconnectStorage observe.Disconnect
}

// OpenService opens the distribution-layer control service using the provided
// configuration(s). Fields defined in each subsequent configuration override those in
// previous configurations.
//
// If the returned error is nil, the Service must be closed after use.
func OpenService(ctx context.Context, cfgs ...ServiceConfig) (*Service, error) {
	cfg, err := config.New(ServiceConfig{}, cfgs...)
	if err != nil {
		return nil, err
	}
	s := &Service{
		cfg:     cfg,
		router:  proxy.BatchFactory[channel.Key](cfg.HostResolver.HostKey()),
		updates: observe.New[Update](),
	}
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
	return s, nil
}

// Retrieve returns the control state of the given channels. Passing no keys returns the
// state of every controlled channel. Channels that no subject controls are absent from
// the result.
func (s *Service) Retrieve(
	ctx context.Context,
	keys ...channel.Key,
) ([]State, error) {
	if len(keys) == 0 {
		return statesFromStorage(s.cfg.TS.ControlStates()), nil
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

// OnChange registers handler to receive every control transfer the host arbitrates. The
// returned function unregisters the handler.
func (s *Service) OnChange(
	handler func(context.Context, Update),
) observe.Disconnect {
	return s.updates.OnChange(handler)
}

// Close releases the service's resources.
func (s *Service) Close() error {
	s.disconnectStorage()
	return s.shutdown.Close()
}

func (s *Service) retrievePeer(
	ctx context.Context,
	target node.Key,
	keys channel.Keys,
) ([]State, error) {
	addr, err := s.cfg.HostResolver.Resolve(target)
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
