package writer

import (
	"context"
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

type StreamWriter = confluence.Segment[Request, Response]

type Writer interface {
	Write(frame core.Frame) bool
	Commit() bool
	Error() error
	Close() error
}

type writer struct {
	requests          confluence.Inlet[Request]
	responses         confluence.Outlet[Response]
	wg                signal.WaitGroup
	shutdown          context.CancelFunc
	hasAccumulatedErr bool
}

func (w *writer) Write(frame core.Frame) bool {
	if w.hasAccumulatedErr {
		return false
	}
	select {
	case <-w.wg.Stopped():
		return false
	case <-w.responses.Outlet():
		w.hasAccumulatedErr = true
		return false
	case w.requests.Inlet() <- Request{Frame: frame}:
		return true
	}
}

func (w *writer) Commit() bool {
	if w.hasAccumulatedErr {
		return false
	}
	select {
	case <-w.wg.Stopped():
		return false
	case w.requests.Inlet() <- Request{Command: Commit}:
	}
	for res := range w.responses.Outlet() {
		if res.Command == Commit {
			return res.Ack
		}
	}
	return false
}

func (w *writer) Error() error {
	w.requests.Inlet() <- Request{Command: Error}
	for res := range w.responses.Outlet() {
		if res.Command == Error {
			return res.Err
		}
	}
	return nil
}

func (w *writer) Close() error {
	w.requests.Close()
	err := w.wg.Wait()
	for range w.responses.Outlet() {
	}
	w.shutdown()
	return err
}

type Config struct {
	Keys            channel.Keys
	Start           telem.TimeStamp
	TS              storage.TS
	ChannelService  *channel.Service
	HostResolver    distribcore.HostResolver
	TransportServer TransportServer
	TransportClient TransportClient
	Logger          *zap.Logger
}

var _ config.Config[Config] = Config{}

func (cfg Config) Override(other Config) Config {
	cfg.TS = override.Nil(cfg.TS, other.TS)
	cfg.ChannelService = override.Nil(cfg.ChannelService, other.ChannelService)
	cfg.HostResolver = override.Nil(cfg.HostResolver, other.HostResolver)
	cfg.TransportServer = override.Nil(cfg.TransportServer, other.TransportServer)
	cfg.TransportClient = override.Nil(cfg.TransportClient, other.TransportClient)
	cfg.Keys = override.Nil(cfg.Keys, other.Keys)
	cfg.Logger = override.Nil(cfg.Logger, other.Logger)
	return cfg
}

func (cfg Config) Validate() error {
	v := validate.New("writerClient")
	validate.NotNil(v, "TS", cfg.TS)
	validate.NotNil(v, "ChannelService", cfg.ChannelService)
	validate.NotNil(v, "HostResolver", cfg.HostResolver)
	validate.NotNil(v, "TransportServer", cfg.TransportServer)
	validate.NotNil(v, "TransportClient", cfg.TransportClient)
	validate.NotEmptySlice(v, "Keys", cfg.Keys)
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
	hostID := cfg.HostResolver.HostID()

	// First we need to check if all the channels exist and are retrievable in the
	//database.
	if err := core.ValidateChannelKeys(
		ctx,
		cfg.ChannelService,
		cfg.Keys,
	); err != nil {
		return nil, err
	}

	// TraverseTo we determine the IDs of all the target nodes we need to write to.
	batch := proxy.NewBatchFactory[channel.Key](hostID).Batch(cfg.Keys)

	var (
		pipe              = plumber.New()
		needRemote        = len(batch.Remote) > 0
		needLocal         = len(batch.Local) > 0
		receiverAddresses []address.Address
	)

	transient := confluence.NewStream[error](0)

	// The synchronizer checks that all nodes have acknowledged an iteration
	// request. This is used to return ok = true from the iterator methods.
	plumber.SetSegment[Response, Response](
		pipe,
		"synchronizer",
		newSynchronizer(len(cfg.Keys.UniqueNodeIDs())),
	)

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

		// SetState up our sender as a sink for the request pipeline.
		plumber.SetSink[Request](pipe, "remoteSender", sender)
		receiverAddresses = _receiverAddresses
		for i, receiver := range receivers {
			plumber.SetSource[Response](pipe, _receiverAddresses[i], receiver)
		}
	}

	if needLocal {
		w, err := newLocalWriter(batch.Local, cfg)
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

	if needRemote {
		receiverAddresses = append(receiverAddresses, "transient")
		plumber.SetSource[Response](pipe, "transient", &TransientSource{transient: transient})
	}

	plumber.MultiRouter[Response]{
		SourceTargets: receiverAddresses,
		SinkTargets:   []address.Address{"synchronizer"},
		Stitch:        plumber.StitchUnary,
		Capacity:      len(receiverAddresses),
	}.MustRoute(pipe)

	seg := &plumber.Segment[Request, Response]{Pipeline: pipe}
	if err := seg.RouteInletTo(routeRequestsTo); err != nil {
		panic(err)
	}
	if err := seg.RouteOutletFrom("synchronizer"); err != nil {
		panic(err)
	}

	input := confluence.NewStream[Request]()
	output := confluence.NewStream[Response]()
	seg.InFrom(input)
	seg.OutTo(output)

	return seg, nil
}
