package writer

import (
	"context"
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	dcore "github.com/synnaxlabs/synnax/pkg/distribution/core"
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
	HostResolver dcore.HostResolver
	// Transport is the network transport for sending and receiving writes from other
	// nodes in the cluster.
	// [REQUIRED]
	Transport Transport
	// Logger is the witness of it all.
	// [OPTIONAL]
	Logger *zap.Logger
}

var (
	_ config.Config[ServiceConfig] = ServiceConfig{}
	// DefaultConfig is the default configuration for opening a Writer or StreamWriter. It
	// is not complete and must be supplemented with the required fields.
	DefaultConfig = ServiceConfig{Logger: zap.NewNop()}
)

// Validate implements config.Config.
func (cfg ServiceConfig) Validate() error {
	v := validate.New("distribution.framer.writer")
	validate.NotNil(v, "TS", cfg.TS)
	validate.NotNil(v, "ChannelReader", cfg.ChannelReader)
	validate.NotNil(v, "HostResolver", cfg.HostResolver)
	validate.NotNil(v, "Transport", cfg.Transport)
	validate.NotNil(v, "Logger", cfg.Logger)
	return v.Error()
}

// Override implements config.Config.
func (cfg ServiceConfig) Override(other ServiceConfig) ServiceConfig {
	cfg.TS = override.Nil(cfg.TS, other.TS)
	cfg.ChannelReader = override.Nil(cfg.ChannelReader, other.ChannelReader)
	cfg.HostResolver = override.Nil(cfg.HostResolver, other.HostResolver)
	cfg.Transport = override.Nil(cfg.Transport, other.Transport)
	cfg.Logger = override.Nil(cfg.Logger, other.Logger)
	return cfg
}

type Service struct {
	ServiceConfig
	server *server
}

func OpenService(configs ...ServiceConfig) (*Service, error) {
	cfg, err := config.OverrideAndValidate(DefaultConfig, configs...)
	return &Service{ServiceConfig: cfg, server: startServer(cfg)}, err
}

const (
	synchronizerAddr      = address.Address("synchronizer")
	peerSenderAddr        = address.Address("peerSender")
	gatewayWriterAddr     = address.Address("gatewayWriter")
	peerGatewaySwitchAddr = address.Address("peerGatewaySwitch")
	bulkheadAddr          = address.Address("bulkhead")
	bulkheadResponsesAddr = address.Address("bulkheadResponses")
)

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
	if err := core.ValidateChannelKeys(ctx, s.ChannelReader, cfg.Keys); err != nil {
		return nil, err
	}

	var (
		hostID             = s.HostResolver.HostID()
		batch              = proxy.NewBatchFactory[channel.Key](hostID).Batch(cfg.Keys)
		pipe               = plumber.New()
		needPeerRouting    = len(batch.Peers) > 0
		needGatewayRouting = len(batch.Gateway) > 0
		receiverAddresses  []address.Address
		routeBulkheadTo    address.Address
	)

	bh := &bulkhead{signal: make(chan bool, 1)}
	plumber.SetSegment[Request, Request](pipe, bulkheadAddr, bh)
	plumber.SetSource[Response](pipe, bulkheadResponsesAddr, &bh.responses)
	plumber.SetSegment[Response, Response](
		pipe,
		synchronizerAddr,
		newSynchronizer(len(cfg.Keys.UniqueNodeIDs()), bh.signal),
	)

	if needPeerRouting {
		routeBulkheadTo = peerSenderAddr
		sender, receivers, _receiverAddresses, err := s.openManyPeers(ctx, batch.Peers)
		if err != nil {
			return nil, err
		}
		plumber.SetSegment[Request, Response](pipe, peerSenderAddr, sender)
		receiverAddresses = _receiverAddresses
		for i, receiver := range receivers {
			plumber.SetSource[Response](pipe, _receiverAddresses[i], receiver)
		}
		receiverAddresses = append(receiverAddresses, peerSenderAddr)
	}

	if needGatewayRouting {
		routeBulkheadTo = gatewayWriterAddr
		w, err := s.newGateway(Config{Start: cfg.Start, Keys: batch.Gateway})
		if err != nil {
			return nil, err
		}
		plumber.SetSegment[Request, Response](pipe, gatewayWriterAddr, w)
		receiverAddresses = append(receiverAddresses, gatewayWriterAddr)
	}

	if needPeerRouting && needGatewayRouting {
		routeBulkheadTo = peerGatewaySwitchAddr
		plumber.SetSegment[Request, Request](
			pipe,
			peerGatewaySwitchAddr,
			newPeerGatewaySwitch(hostID),
		)
		plumber.MultiRouter[Request]{
			SourceTargets: []address.Address{peerGatewaySwitchAddr},
			SinkTargets:   []address.Address{peerSenderAddr, gatewayWriterAddr},
			Stitch:        plumber.StitchWeave,
			Capacity:      2,
		}.MustRoute(pipe)
	}

	plumber.MustConnect[Request](pipe, bulkheadAddr, routeBulkheadTo, 1)

	plumber.MultiRouter[Response]{
		SourceTargets: receiverAddresses,
		SinkTargets:   []address.Address{synchronizerAddr},
		Stitch:        plumber.StitchUnary,
		Capacity:      len(receiverAddresses),
	}.MustRoute(pipe)

	seg := &plumber.Segment[Request, Response]{Pipeline: pipe}
	lo.Must0(seg.RouteInletTo(bulkheadAddr))
	lo.Must0(seg.RouteOutletFrom(bulkheadResponsesAddr, synchronizerAddr))
	return seg, nil
}
