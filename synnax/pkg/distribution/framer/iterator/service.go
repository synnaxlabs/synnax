package iterator

import (
	"context"
	"github.com/samber/lo"
	"github.com/synnaxlabs/aspen"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/proxy"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

type Config struct {
	Bounds telem.TimeRange
	Keys   channel.Keys
}

type ServiceConfig struct {
	TS            storage.StreamIterableTS
	ChannelReader channel.Reader
	HostResolver  aspen.HostResolver
	Transport     Transport
	Logger        *zap.Logger
}

var _ config.Config[ServiceConfig] = ServiceConfig{}

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
	validate.NotNil(v, "TSChannel", cfg.TS)
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

var DefaultConfig = ServiceConfig{
	Logger: zap.NewNop(),
}

func NewService(configs ...ServiceConfig) (*Service, error) {
	cfg, err := config.OverrideAndValidate(DefaultConfig, configs...)
	return &Service{
		ServiceConfig: cfg,
		server:        startServer(cfg),
	}, err
}

func (s *Service) New(ctx context.Context, cfg Config) (Iterator, error) {
	stream, err := s.NewStream(ctx, cfg)
	if err != nil {
		return nil, err
	}
	sCtx, cancel := signal.Background(
		signal.WithLogger(s.Logger),
		signal.WithContextKey("iterator"),
	)
	requests := confluence.NewStream[Request]()
	responses := confluence.NewStream[Response]()
	stream.InFrom(requests)
	stream.OutTo(responses)
	stream.Flow(
		sCtx,
		confluence.CloseInletsOnExit(),
		confluence.CancelOnExitErr(),
	)
	return &iterator{
		requests:  requests,
		responses: responses,
		internal:  stream,
		shutdown:  cancel,
		wg:        sCtx,
	}, nil
}

func (s *Service) NewStream(ctx context.Context, cfg Config) (StreamIterator, error) {
	// First we need to check if all the channels exist and are retrievable in the
	// database.
	if err := core.ValidateChannelKeys(ctx, s.ChannelReader, cfg.Keys); err != nil {
		return nil, err
	}

	// Determine IDs of all the target nodes we need to open iterators on.
	batch := proxy.NewBatchFactory[channel.Key](s.HostResolver.HostID()).Batch(cfg.Keys)

	var (
		pipe              = plumber.New()
		needRemote        = len(batch.Peers) > 0
		needLocal         = len(batch.Gateway) > 0
		receiverAddresses []address.Address
	)

	if needRemote {
		sender, receivers, err := s.openManyPeers(ctx, cfg.Bounds, batch.Peers)
		if err != nil {
			return nil, err
		}

		// SetState up our sender as a sink for the request pipeline.
		plumber.SetSink[Request](pipe, "sender", sender)

		// SetState up our remote receivers as sources for the response pipeline.
		receiverAddresses = make([]address.Address, len(receivers))
		for i, c := range receivers {
			addr := address.Newf("client-%v", i+1)
			receiverAddresses[i] = addr
			plumber.SetSource[Response](pipe, addr, c)
		}
	}

	if needLocal {
		gwCfg := Config{Keys: batch.Gateway, Bounds: cfg.Bounds}
		gatewwayIter, err := newStorageIterator(s.ServiceConfig, gwCfg)
		if err != nil {
			return nil, err
		}
		addr := address.Address("gateway")
		plumber.SetSegment[Request, Response](pipe, addr, gatewwayIter)
		receiverAddresses = append(receiverAddresses, addr)
	}

	plumber.SetSegment[Response, Response](pipe, "filter", newAckFilter())

	// The synchronizer checks that all nodes have acknowledged an iteration
	// request. This is used to return ok = true from the iterator methods.
	plumber.SetSegment[Response, Response](
		pipe,
		"synchronizer",
		newSynchronizer(len(cfg.Keys.UniqueNodeIDs())),
	)

	var routeInletTo address.Address

	// We need to configure different pipelines to optimize for particular cases.
	if needRemote && needLocal {
		// Open a broadcaster that will multiply requests to both the local and remote
		// iterators.
		plumber.SetSegment[Request, Request](
			pipe,
			"broadcaster",
			&confluence.DeltaMultiplier[Request]{},
		)
		routeInletTo = "broadcaster"

		// We use confluence.StitchWeave here to dedicate a channelClient to both the
		// sender and local, so that they both receive a copy of the emitted request.
		plumber.MultiRouter[Request]{
			SourceTargets: []address.Address{"broadcaster"},
			SinkTargets:   []address.Address{"sender", "gateway"},
			Stitch:        plumber.StitchWeave,
		}.MustRoute(pipe)
	} else if needRemote {
		// If we only have remote iterators, we can skip the broadcasting step
		// and forward requests from the emitter directly to the sender.
		routeInletTo = "sender"
	} else {
		// If we only have a gateway iterator, we can skip the broadcasting step
		// and forward requests from the emitter directly to the local iterator.
		routeInletTo = "gateway"
	}

	plumber.MultiRouter[Response]{
		SourceTargets: receiverAddresses,
		SinkTargets:   []address.Address{"filter"},
		Stitch:        plumber.StitchUnary,
		Capacity:      len(receiverAddresses),
	}.MustRoute(pipe)

	plumber.UnaryRouter[Response]{
		SourceTarget: "filter",
		SinkTarget:   "synchronizer",
	}.MustRoute(pipe)

	seg := &plumber.Segment[Request, Response]{Pipeline: pipe}
	lo.Must0(seg.RouteOutletFrom("filter", "synchronizer"))
	lo.Must0(seg.RouteInletTo(routeInletTo))

	return seg, nil
}
