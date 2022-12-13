package iterator

import (
	"context"
	"github.com/cockroachdb/errors"
	"github.com/samber/lo"
	"github.com/synnaxlabs/aspen"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/proxy"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

type Config struct {
	Keys   channel.Keys
	Bounds telem.TimeRange
}

type ServiceConfig struct {
	TS            storage.StreamIterableTS
	ChannelReader channel.Reader
	HostResolver  aspen.HostResolver
	Transport     Transport
	Logger        *zap.Logger
}

var (
	_             config.Config[ServiceConfig] = ServiceConfig{}
	DefaultConfig                              = ServiceConfig{Logger: zap.NewNop()}
)

// Override implements Config.
func (cfg ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	cfg.TS = override.Nil(cfg.TS, other.TS)
	cfg.ChannelReader = override.Nil(cfg.ChannelReader, other.ChannelReader)
	cfg.Transport = override.Nil(cfg.Transport, other.Transport)
	cfg.HostResolver = override.Nil(cfg.HostResolver, other.HostResolver)
	cfg.Logger = override.Nil(cfg.Logger, other.Logger)
	return cfg
}

// Validate implements Config.
func (cfg ServiceConfig) Validate() error {
	v := validate.New("distribution.framer.iterator")
	validate.NotNil(v, "TS", cfg.TS)
	validate.NotNil(v, "ChannelReader", cfg.ChannelReader)
	validate.NotNil(v, "Transport", cfg.Transport)
	validate.NotNil(v, "Resolver", cfg.HostResolver)
	validate.NotNil(v, "Logger", cfg.Logger)
	return v.Error()
}

type Service struct {
	ServiceConfig
	server *server
}

func OpenService(configs ...ServiceConfig) (*Service, error) {
	cfg, err := config.OverrideAndValidate(DefaultConfig, configs...)
	return &Service{
		ServiceConfig: cfg,
		server:        startServer(cfg),
	}, err
}

const (
	peerSenderAddr   address.Address = "peerSender"
	gatewayIterAddr  address.Address = "gatewayWriter"
	broadcasterAddr  address.Address = "broadcaster"
	synchronizerAddr address.Address = "synchronizer"
)

func (s *Service) New(ctx context.Context, cfg Config) (Iterator, error) {
	stream, err := s.NewStream(ctx, cfg)
	if err != nil {
		return nil, err
	}
	sCtx, cancel := signal.Background(
		signal.WithLogger(s.Logger),
		signal.WithContextKey("iterator"),
	)
	req := confluence.NewStream[Request]()
	res := confluence.NewStream[Response]()
	stream.InFrom(req)
	stream.OutTo(res)
	stream.Flow(sCtx, confluence.CloseInletsOnExit(), confluence.CancelOnExitErr())
	return &iterator{requests: req, responses: res, shutdown: cancel, wg: sCtx}, nil
}

func (s *Service) NewStream(ctx context.Context, cfg Config) (StreamIterator, error) {
	if err := s.validateChannelKeys(ctx, cfg.Keys); err != nil {
		return nil, err
	}

	var (
		hostID             = s.HostResolver.HostID()
		batch              = proxy.NewBatchFactory[channel.Key](hostID).Batch(cfg.Keys)
		pipe               = plumber.New()
		needPeerRouting    = len(batch.Peers) > 0
		needGatewayRouting = len(batch.Gateway) > 0
		receiverAddresses  []address.Address
		routeInletTo       address.Address
	)

	if needPeerRouting {
		routeInletTo = peerSenderAddr
		sender, receivers, err := s.openManyPeers(ctx, cfg.Bounds, batch.Peers)
		if err != nil {
			return nil, err
		}
		plumber.SetSink[Request](pipe, peerSenderAddr, sender)
		receiverAddresses = make([]address.Address, len(receivers))
		for i, c := range receivers {
			addr := address.Newf("client-%v", i+1)
			receiverAddresses[i] = addr
			plumber.SetSource[Response](pipe, addr, c)
		}
	}

	if needGatewayRouting {
		routeInletTo = gatewayIterAddr
		gatewayIter, err := s.newGateway(Config{Keys: batch.Gateway, Bounds: cfg.Bounds})
		if err != nil {
			return nil, err
		}
		plumber.SetSegment[Request, Response](pipe, gatewayIterAddr, gatewayIter)
		receiverAddresses = append(receiverAddresses, gatewayIterAddr)
	}

	if needPeerRouting && needGatewayRouting {
		routeInletTo = broadcasterAddr
		plumber.SetSegment[Request, Request](
			pipe,
			broadcasterAddr,
			&confluence.DeltaMultiplier[Request]{},
		)
		plumber.MultiRouter[Request]{
			SourceTargets: []address.Address{broadcasterAddr},
			SinkTargets:   []address.Address{peerSenderAddr, gatewayIterAddr},
			Stitch:        plumber.StitchWeave,
			Capacity:      2,
		}.MustRoute(pipe)
	}

	plumber.SetSegment[Response, Response](
		pipe,
		synchronizerAddr,
		newSynchronizer(len(cfg.Keys.UniqueNodeIDs())),
	)

	plumber.MultiRouter[Response]{
		SourceTargets: receiverAddresses,
		SinkTargets:   []address.Address{synchronizerAddr},
		Stitch:        plumber.StitchUnary,
		Capacity:      len(receiverAddresses),
	}.MustRoute(pipe)

	seg := &plumber.Segment[Request, Response]{Pipeline: pipe}
	lo.Must0(seg.RouteOutletFrom(synchronizerAddr))
	lo.Must0(seg.RouteInletTo(routeInletTo))
	return seg, nil
}

func (s *Service) validateChannelKeys(ctx context.Context, keys channel.Keys) error {
	v := validate.New("distribution.framer.iterator")
	if validate.NotEmptySlice(v, "Keys", keys) {
		return v.Error()
	}
	exists, err := s.ChannelReader.NewRetrieve().WhereKeys(keys...).Exists(ctx)
	if err != nil {
		return err
	}
	if !exists {
		return errors.Wrapf(query.NotFound, "some channel keys %v not found", keys)
	}
	return nil
}
