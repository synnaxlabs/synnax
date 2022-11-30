package writer

import (
	"context"
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	distribcore "github.com/synnaxlabs/synnax/pkg/distribution/core"
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
	// Keys is the set of keys to write to. Every frame must contain an array for every
	// key.
	// [REQUIRED]
	Keys channel.Keys
	// Start marks the starting timestamp of the first sample in the first frame. If
	// telemetry occupying the given timestamp already exists for the provided channels,
	// the writer will fail to open.
	// [REQUIRED]
	Start telem.TimeStamp
}

// ServiceConfig is the configuration for opening a Writer or StreamWriter.
type ServiceConfig struct {
	// TS is the local time series store to write to.
	// [REQUIRED]
	TS storage.StreamWritableTS
	// ChannelReader is used to resolve metadata and routing information for the provided
	// channels.
	// [REQUIRED]
	ChannelReader channel.Reader
	// HostResolver is used to resolve the host address for nodes in the cluster in order
	// to route writes.
	// [REQUIRED]
	HostResolver distribcore.HostResolver
	// Transport is the network transport for sending and receiving writes from other
	// nodes in the cluster.
	// [REQUIRED]
	Transport Transport
	// Logger is the witness of it all.
	// [OPTIONAL]
	Logger *zap.Logger
}

// DefaultConfig is the default configuration for opening a Writer or StreamWriter. It
// is not complete and must be supplemented with the required fields.
var DefaultConfig = ServiceConfig{Logger: zap.NewNop()}

var _ config.Config[ServiceConfig] = ServiceConfig{}

// Override implements ServiceConfig.
func (cfg ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	cfg.TS = override.Nil(cfg.TS, other.TS)
	cfg.ChannelReader = override.Nil(cfg.ChannelReader, other.ChannelReader)
	cfg.HostResolver = override.Nil(cfg.HostResolver, other.HostResolver)
	cfg.Transport = override.Nil(cfg.Transport, other.Transport)
	cfg.Logger = override.Nil(cfg.Logger, other.Logger)
	return cfg
}

// Validate implements ServiceConfig.
func (cfg ServiceConfig) Validate() error {
	v := validate.New("writerClient")
	validate.NotNil(v, "TS", cfg.TS)
	validate.NotNil(v, "ChannelReader", cfg.ChannelReader)
	validate.NotNil(v, "HostResolver", cfg.HostResolver)
	validate.NotNil(v, "Transport", cfg.Transport)
	validate.NotNil(v, "logger", cfg.Logger)
	return v.Error()
}

type Service struct {
	ServiceConfig
	server *server
}

func NewService(configs ...ServiceConfig) (*Service, error) {
	cfg, err := config.OverrideAndValidate(DefaultConfig, configs...)
	return &Service{ServiceConfig: cfg, server: startServer(cfg)}, err
}

func (s *Service) New(ctx context.Context, cfg Config) (Writer, error) {
	sCtx, cancel := signal.WithCancel(
		ctx,
		signal.WithContextKey("writer"),
		signal.WithLogger(s.Logger),
	)
	seg, err := s.NewStream(ctx, cfg)
	if err != nil {
		return nil, err
	}
	req := confluence.NewStream[Request]()
	res := confluence.NewStream[Response]()
	seg.InFrom(req)
	seg.OutTo(res)
	seg.Flow(sCtx, confluence.CloseInletsOnExit(), confluence.CancelOnExitErr())
	return &writer{requests: req, responses: res, wg: sCtx, shutdown: cancel}, nil
}

// NewStream opens a new StreamWriter using the given configuration. The provided context
// is only used for opening the stream and is not used for concurrent flow control.
func (s *Service) NewStream(ctx context.Context, cfg Config) (StreamWriter, error) {
	hostID := s.HostResolver.HostID()

	// Assert all the channels exist and are retrievable in the cluster.
	if err := core.ValidateChannelKeys(ctx, s.ChannelReader, cfg.Keys); err != nil {
		return nil, err
	}

	// Determine the IDs of all the target nodes we need to write to.
	batch := proxy.NewBatchFactory[channel.Key](hostID).Batch(cfg.Keys)

	var (
		pipe               = plumber.New()
		needPeerRouting    = len(batch.Peers) > 0
		needGatewayRouting = len(batch.Gateway) > 0
		receiverAddresses  []address.Address
	)

	bulkhead := newBulkhead()
	bulkhead.signal = make(chan bool, 1)

	plumber.SetSegment[Request, Request](
		pipe,
		"bulkhead",
		bulkhead,
	)

	plumber.SetSource[Response](
		pipe,
		"bulkheadResponses",
		&bulkhead.responses,
	)

	// The synchronizer checks that all nodes have acknowledged an iteration
	// request. This is used to return ok = true from the iterator methods.
	plumber.SetSegment[Response, Response](
		pipe,
		"synchronizer",
		newSynchronizer(len(cfg.Keys.UniqueNodeIDs())),
	)

	if needPeerRouting {
		sender, receivers, _receiverAddresses, err := s.openManyPeers(ctx, batch.Peers)
		if err != nil {
			return nil, err
		}

		// Set up our sender as a sink for the request pipeline.
		plumber.SetSegment[Request, Response](pipe, "peerSender", sender)
		receiverAddresses = _receiverAddresses
		for i, receiver := range receivers {
			plumber.SetSource[Response](pipe, _receiverAddresses[i], receiver)
		}
		receiverAddresses = append(receiverAddresses, "peerSender")
	}

	if needGatewayRouting {
		gwCfg := Config{Start: cfg.Start, Keys: batch.Gateway}
		w, err := s.newGatewayWriter(gwCfg)
		if err != nil {
			return nil, err
		}
		addr := address.Address("localWriter")
		plumber.SetSegment[Request, Response](pipe, addr, w)
		receiverAddresses = append(receiverAddresses, addr)
	}

	var routeBulkheadTo address.Address

	if needPeerRouting && needGatewayRouting {
		rls := newPeerGatewaySwitch(hostID)
		plumber.SetSegment[Request, Request](pipe, "peerGatewaySwitch", rls)
		routeBulkheadTo = "peerGatewaySwitch"
		plumber.MultiRouter[Request]{
			SourceTargets: []address.Address{"peerGatewaySwitch"},
			SinkTargets:   []address.Address{"peerSender", "localWriter"},
			Stitch:        plumber.StitchWeave,
			Capacity:      2,
		}.MustRoute(pipe)
	} else if needPeerRouting {
		routeBulkheadTo = "peerSender"
	} else {
		routeBulkheadTo = "localWriter"
	}

	plumber.MustConnect(pipe, "bulkhead", routeBulkheadTo, 1)

	plumber.MultiRouter[Response]{
		SourceTargets: receiverAddresses,
		SinkTargets:   []address.Address{"synchronizer"},
		Stitch:        plumber.StitchUnary,
		Capacity:      len(receiverAddresses),
	}.MustRoute(pipe)

	seg := &plumber.Segment[Request, Response]{Pipeline: pipe}
	lo.Must0(seg.RouteInletTo("bulkhead"))
	lo.Must0(seg.RouteOutletFrom("bulkheadResponses", "synchronizer"))
	return seg, nil
}
