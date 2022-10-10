package writer

import (
	"context"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	distribcore "github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/proxy"
	"github.com/synnaxlabs/synnax/pkg/distribution/segment/core"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
)

type StreamWriter = confluence.Segment[Request, Response]

type Writer interface {
	Write([]core.Segment) bool
	Close() error
}

type writer struct {
	requests  confluence.Inlet[Request]
	responses confluence.Outlet[Response]
	wg        signal.WaitGroup
	shutdown  context.CancelFunc
	error     error
}

func (w *writer) Write(segments []core.Segment) bool {
	if w.error != nil {
		return false
	}
	w.requests.Inlet() <- Request{Segments: segments}
	select {
	case <-w.wg.Stopped():
		return false
	case res := <-w.responses.Outlet():
		w.error = res.Err
		return false
	default:
		return true
	}
}

func (w *writer) Close() error {
	w.requests.Close()
	err := w.wg.Wait()
	for res := range w.responses.Outlet() {
		if res.Err != nil {
			err = res.Err
		}
	}
	w.shutdown()
	return err
}

type Config struct {
	TS              storage.TS
	ChannelService  *channel.Service
	Resolver        distribcore.HostResolver
	TransportServer TransportServer
	TransportClient TransportClient
	ChannelKeys     channel.Keys
	Logger          *zap.Logger
}

func (cfg Config) Override(other Config) Config {
	cfg.TS = override.Nil(cfg.TS, other.TS)
	cfg.ChannelService = override.Nil(cfg.ChannelService, other.ChannelService)
	cfg.Resolver = override.Nil(cfg.Resolver, other.Resolver)
	cfg.TransportServer = override.Nil(cfg.TransportServer, other.TransportServer)
	cfg.TransportClient = override.Nil(cfg.TransportClient, other.TransportClient)
	cfg.ChannelKeys = override.Nil(cfg.ChannelKeys, other.ChannelKeys)
	cfg.Logger = override.Nil(cfg.Logger, other.Logger)
	return cfg
}

func (cfg Config) Validate() error {
	v := validate.New("writerClient")
	validate.NotNil(v, "TS", cfg.TS)
	validate.NotNil(v, "ChannelService", cfg.ChannelService)
	validate.NotNil(v, "Resolver", cfg.Resolver)
	validate.NotNil(v, "TransportServer", cfg.TransportServer)
	validate.NotNil(v, "TransportClient", cfg.TransportClient)
	validate.NotEmptySlice(v, "ChannelKeys", cfg.ChannelKeys)
	validate.NotNil(v, "logger", cfg.Logger)
	return v.Error()
}

func New(ctx context.Context, cfg Config) (Writer, error) {
	sCtx, cancel := signal.WithCancel(
		ctx,
		signal.WithContextKey("writerClient"),
		signal.WithLogger(cfg.Logger),
	)
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	seg, err := NewStream(ctx, cfg)
	if err != nil {
		return nil, err
	}
	requests := confluence.NewStream[Request]()
	responses := confluence.NewStream[Response]()
	seg.InFrom(requests)
	seg.OutTo(responses)
	seg.Flow(sCtx, confluence.CloseInletsOnExit(), confluence.CancelOnExitErr())
	return &writer{
		requests:  requests,
		responses: responses,
		wg:        sCtx,
		shutdown:  cancel,
	}, nil
}

func NewStream(ctx context.Context, _cfg ...Config) (StreamWriter, error) {
	cfg, err := config.OverrideAndValidate(Config{}, _cfg...)
	if err != nil {
		return nil, err
	}
	hostID := cfg.Resolver.HostID()

	// First we need to check if all the channels exist and are retrievable in the
	//database.
	if err := core.ValidateChannelKeys(
		ctx,
		cfg.ChannelService,
		cfg.ChannelKeys,
	); err != nil {
		return nil, err
	}

	// TraverseTo we determine the IDs of all the target nodes we need to write to.
	batch := proxy.NewBatchFactory[channel.Key](hostID).Batch(cfg.ChannelKeys)

	var (
		pipe              = plumber.New()
		needRemote        = len(batch.Remote) > 0
		needLocal         = len(batch.Local) > 0
		receiverAddresses []address.Address
	)

	transient := confluence.NewStream[error](0)

	if needRemote {
		sender, receivers, _receiverAddresses, err := openRemoteWriters(
			ctx,
			batch.Remote,
			transient,
			cfg,
		)
		if err != nil {
			return nil, err
		}

		// Set up our sender as a sink for the request pipeline.
		plumber.SetSink[Request](pipe, "remoteSender", sender)
		receiverAddresses = _receiverAddresses
		for i, receiver := range receivers {
			plumber.SetSource[Response](pipe, _receiverAddresses[i], receiver)
		}
	}

	if needLocal {
		w, err := newLocalWriter(ctx, batch.Local, transient, cfg)
		if err != nil {
			return nil, err
		}
		addr := address.Address("localWriter")
		plumber.SetSegment[Request, Response](pipe, addr, w)
		receiverAddresses = append(receiverAddresses, addr)
	}

	var routeRequestsTo address.Address

	if needRemote && needLocal {
		rls := newRemoteLocalSwitch(hostID)
		plumber.SetSegment[Request, Request](pipe, "remoteLocalSwitch", rls)
		routeRequestsTo = "remoteLocalSwitch"

		if err := (plumber.MultiRouter[Request]{
			SourceTargets: []address.Address{"remoteLocalSwitch"},
			SinkTargets:   []address.Address{"remoteSender", "localWriter"},
			Stitch:        plumber.StitchWeave,
		}).Route(pipe); err != nil {
			panic(err)
		}
	} else if needRemote {
		routeRequestsTo = "remoteSender"
	} else {
		routeRequestsTo = "localWriter"
	}

	receiverAddresses = append(receiverAddresses, "transient")
	plumber.SetSource[Response](pipe, "transient", &TransientSource{transient: transient})

	seg := &plumber.Segment[Request, Response]{Pipeline: pipe}
	if err := seg.RouteInletTo(routeRequestsTo); err != nil {
		panic(err)
	}
	if err := seg.RouteOutletFrom(receiverAddresses...); err != nil {
		panic(err)
	}

	input := confluence.NewStream[Request]()
	output := confluence.NewStream[Response]()
	seg.InFrom(input)
	seg.OutTo(output)

	return seg, nil
}
